package p2c

import (
	"emicro/internal/codes"
	"emicro/internal/utils/xsync"
	"emicro/internal/utils/xtime"
	"fmt"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// This package is a client load loadbalance. The algorithm used is p2c+ewma
// P2c is either
// Ewma index moving weighted average (reflecting the average value over a period of time)

const (
	// Name is the name of p2c loadbalance.
	Name            = "p2c_ewma"
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	// if statistic not collected,we add a big lag penalty to endpoint
	penalty   = int64(math.MaxInt32)
	forcePick = int64(time.Second)
	pickTimes = 3
	// default value from finagle
	decayTime   = int64(time.Second * 10)
	logInterval = time.Minute
)

var emptyPickResult balancer.PickResult

type PickerBuilder struct{}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var conns []*Conn
	for conn, connInfo := range readySCs {
		conns = append(conns, &Conn{
			SubConn: conn,
			success: initSuccess,
			address: connInfo.Address,
		})
	}
	return &Picker{
		conns: conns,
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type Picker struct {
	conns   []*Conn
	r       *rand.Rand
	lock    sync.Mutex
	stamp   *xsync.AtomicDuration
	logFunc func(info string, args ...any)
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var chosen *Conn
	switch len(p.conns) {
	case 0:
		return emptyPickResult, balancer.ErrNoSubConnAvailable
	case 1:
		chosen = p.choose(p.conns[0], nil)
	case 2:
		chosen = p.choose(p.conns[0], p.conns[1])
	default:
		var node1, node2 *Conn
		for i := 0; i < pickTimes; i++ {
			idx1 := p.r.Intn(len(p.conns))
			idx2 := p.r.Intn(len(p.conns) - 1)
			if idx2 >= idx1 {
				idx2++
			}
			node1 = p.conns[idx1]
			node2 = p.conns[idx2]
			if node1.healthy() && node2.healthy() {
				break
			}
		}
		chosen = p.choose(node1, node2)
	}

	atomic.AddInt64(&chosen.inflight, 1)
	atomic.AddInt64(&chosen.requests, 1)

	return balancer.PickResult{
		SubConn: chosen.SubConn,
		Done:    p.buildCallback(chosen),
	}, nil
}

func (p *Picker) choose(c1, c2 *Conn) *Conn {
	start := int64(xtime.Now())
	if c2 == nil {
		atomic.StoreInt64(&c1.pick, start)
		return c1
	}
	if c1.load() > c2.load() {
		c1, c2 = c2, c1
	}
	// If the failed node has never been selected once during forceGap (forcePick), it is forced to be selected once
	// Take advantage of forced opportunities to trigger updates of success rate and delay
	pick := atomic.LoadInt64(&c2.pick)
	if start-pick > forcePick && atomic.CompareAndSwapInt64(&c2.pick, pick, start) {
		return c2
	}
	atomic.StoreInt64(&c1.pick, start)
	return c1

}

func (p *Picker) buildCallback(c *Conn) func(info balancer.DoneInfo) {
	// call time
	start := int64(xtime.Now())
	return func(info balancer.DoneInfo) {
		atomic.AddInt64(&c.inflight, -1)
		// call completion time
		now := xtime.Now()
		// get the last completion time
		last := atomic.SwapInt64(&c.last, int64(now))
		// calculate time interval
		td := int64(now) - last
		if td < 0 {
			td = 0
		}
		// get call delay
		latency := int64(now) - start
		if latency < 0 {
			// request is completed without taking time, which is not reasonably possible
			latency = 0
		}
		// get the last call delay and the time decay coefficient w
		var w float64
		oldLatency := atomic.LoadUint64(&c.latency)
		if oldLatency != 0 {
			// this calculation formula is the attenuation function model in Newton's law
			w = math.Exp(float64(-td) / float64(decayTime))
		}
		// latest delay data calculated according to * EWMA (exponentially weighted moving average algorithm) *
		atomic.StoreUint64(&c.latency, uint64(float64(oldLatency)*w+float64(latency)*(1-w)))
		// the calculation logic of success is the same as above
		success := initSuccess
		if info.Err != nil && !codes.Acceptable(info.Err) {
			success = 0
		}
		oldSuccess := atomic.LoadUint64(&c.success)
		atomic.StoreUint64(&c.success, uint64(float64(oldSuccess)*w+float64(success)*(1-w)))

	}
}

func (p *Picker) logStats() {
	var stats []string

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.conns {
		stats = append(stats, fmt.Sprintf("conn: %s, load: %d, reqs: %d",
			conn.address.Addr, conn.load(), atomic.SwapInt64(&conn.requests, 0)))
	}

	p.logFunc("p2c - %s", strings.Join(stats, "; "))
}

type Conn struct {
	balancer.SubConn
	address resolver.Address
	// client statistic data
	latency uint64
	success uint64

	inflight int64
	// request number in a period time
	requests int64
	// last lastPick timestamp
	last int64
	pick int64
}

func (c *Conn) healthy() bool {
	return atomic.LoadUint64(&c.success) > throttleSuccess
}

// load() calculates the load of the node
// The formula for calculating the load is: load = Sqrt(ewma) * inflight;
// Here's a simple explanation: ewma is equivalent to the average request time,
// and inflight is the number of requests being processed by the current node,
// which is roughly calculated by multiplying the network load of the current node
func (c *Conn) load() int64 {
	// Add 1 to avoid zero
	latency := int64(math.Sqrt(float64(atomic.LoadUint64(&c.latency) + 1)))
	load := latency * (atomic.LoadInt64(&c.inflight) + 1)
	if load == 0 {
		// penalty is the penalty value when there is no data when the node is just started.
		// The default value is max int32
		return penalty
	}
	return load
}
