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

	}
	return balancer.PickResult{
		SubConn: chosen.SubConn,
		Done:    p.buildCallback(chosen),
	}, nil
}

func (p *Picker) choose(c1, c2 *Conn) *Conn {
	//TODO implement me
	panic("implement me")
}

func (p *Picker) buildCallback(c *Conn) func(info balancer.DoneInfo) {
	//TODO implement me
	panic("implement me")
}

func (p *Picker) logStats() {
	//TODO implement me
	panic("implement me")
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
