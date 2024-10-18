package p2c

import (
	"emicro/internal/utils/xsync"
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"math/rand"
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

var (
	_               balancer.Picker    = (*Picker)(nil)
	_               base.PickerBuilder = (*PickerBuilder)(nil)
	emptyPickResult balancer.PickResult
)

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]*Conn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		connections = append(connections, &Conn{
			SubConn: con,
			address: conInfo.Address,
			name:    conInfo.Address.Addr,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &Picker{
		connections: connections,
		filter:      filter,
	}
}

type Picker struct {
	connections []*Conn
	r           *rand.Rand
	lock        sync.Mutex
	stamp       *xsync.AtomicDuration
	logFunc     func(info string, args ...any)
	filter      loadbalance.Filter
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	//TODO implement me
	panic("implement me")
}

type Conn struct {
	balancer.SubConn
	address resolver.Address
	// client statistic data
	// average call delay (used to save ewma value)
	latency uint64
	// request success number
	success uint64
	// the number of requests currently being processed by the node
	inflight int64
	// total number of requests
	requests int64
	// last request completion time, used to calculate ewma value
	last int64
	// last selected time point
	pick int64
	name string
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
