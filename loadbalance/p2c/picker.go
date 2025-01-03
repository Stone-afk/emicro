package p2c

import (
	"emicro/internal/codes"
	"emicro/internal/utils/xsync"
	"emicro/internal/utils/xtime"
	"emicro/loadbalance"
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

// This package is a client load balancer. The algorithm used is p2c+ewma;
// p2c is the "power of two choices" algorithm,
// ewma is the exponentially weighted moving average (reflects the average over a period of time).
// 该包是一个客户端负载均衡器。使用的算法是 p2c+ewma；
// p2c 是二选一算法（power of two choices）
// ewma 是指数移动加权平均（反映一段时间内的平均值）；

const (
	// Name is the name of the p2c load balancing algorithm
	// Name 是 p2c 负载均衡算法的名称
	Name            = "p2c_ewma"
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	// If no statistical data is collected, we add a large latency penalty to this endpoint.
	// 如果未收集到统计数据，我们会对该端点添加一个较大的延迟惩罚
	penalty   = int64(math.MaxInt32)
	forcePick = int64(time.Second)
	pickTimes = 3
	// 默认值来自 Finagle
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
			success: initSuccess,
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
		filter:      filter,
		connections: connections,
		stamp:       xsync.NewAtomicDuration(),
		r:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *PickerBuilder) Name() string {
	return Name
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
	p.lock.Lock()
	defer p.lock.Unlock()
	var chosen *Conn
	switch len(p.connections) {
	case 0:
		return emptyPickResult, balancer.ErrNoSubConnAvailable
	case 1:
		chosen = p.choose(p.connections[0], nil)
	case 2:
		chosen = p.choose(p.connections[0], p.connections[1])
	default:
		var node1, node2 *Conn
		for i := 0; i < pickTimes; i++ {
			idx1 := p.r.Intn(len(p.connections))
			idx2 := p.r.Intn(len(p.connections) - 1)
			if idx2 >= idx1 {
				idx2++
			}
			node1 = p.connections[idx1]
			node2 = p.connections[idx2]
			if node1.healthy() && node2.healthy() {
				break
			}
		}
		chosen = p.choose(node1, node2)
	}
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
	// 如果失败的节点（负载更大的节点）在 forceGap（forcePick）期间从未被选中过，则强制选择一次
	// 利用强制选择的机会触发成功率和延迟的更新
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
	atomic.AddInt64(&c.inflight, 1)
	atomic.AddInt64(&c.requests, 1)
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

		stamp := p.stamp.Load()
		if now-stamp >= logInterval {
			if p.stamp.CompareAndSwap(stamp, now) {
				p.logStats()
			}
		}
	}
}

func (p *Picker) logStats() {
	var stats []string
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, conn := range p.connections {
		stats = append(stats, fmt.Sprintf("conn: %s, load: %d, reqs: %d",
			conn.address.Addr, conn.load(), atomic.SwapInt64(&conn.requests, 0)))
	}
	p.logFunc("p2c - %s", strings.Join(stats, "; "))
}

type Conn struct {
	balancer.SubConn
	address resolver.Address
	// Client statistics
	// Average call latency (used to store the ewma value)
	// 客户端统计数据
	// 平均调用延迟（用于保存 ewma 值）
	latency uint64
	// Number of successful requests
	// 请求成功数
	success uint64
	// Number of requests currently being processed by the node
	// 节点当前正在处理的请求数
	inflight int64
	// Total number of requests
	// 请求总数
	requests int64
	// Time of last request completion, used to calculate the ewma value
	// 上次请求完成时间，用于计算ewma值
	last int64
	// Time of last selection
	// 上次选择的时间点
	pick int64
	// Node name
	// 节点名称
	name string
}

func (c *Conn) healthy() bool {
	return atomic.LoadUint64(&c.success) > throttleSuccess
}

// load() calculates the load of the node
// The load calculation formula is: load = Sqrt(ewma) * inflight;
// Simple explanation: ewma is the equivalent of the average request time,
// inflight is the number of requests currently being processed by the node,
// roughly calculated by multiplying the network load of the current node.

// load() 计算节点的负载
// 负载计算公式为：load = Sqrt(ewma) * inflight；
// 简单解释：ewma 相当于平均请求时间，
// inflight 是当前节点正在处理的请求数量，
// 这个数量大致通过乘以当前节点的网络负载来计算。
func (c *Conn) load() int64 {
	// Add 1 to avoid zero
	latency := int64(math.Sqrt(float64(atomic.LoadUint64(&c.latency) + 1)))
	load := latency * (atomic.LoadInt64(&c.inflight) + 1)
	if load == 0 {
		// penalty is the penalty value when there is no data when the node is just started.
		// The default value is max int32
		// penalty 是节点刚启动时没有数据时的惩罚值。
		// 默认值为 int32 的最大值
		return penalty
	}
	return load
}
