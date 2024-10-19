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

// 该包是一个客户端负载均衡器。使用的算法是 p2c+ewma；
// p2c 是二选一算法（power of two choices）
// ewma 是指数移动加权平均（反映一段时间内的平均值）；

const (
	// Name 是 p2c 负载均衡算法的名称
	Name            = "p2c_ewma"
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
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
	// 客户端统计数据
	// 平均调用延迟（用于保存 ewma 值）
	latency uint64
	// 请求成功数
	success uint64
	// 节点当前正在处理的请求数
	inflight int64
	// 请求总数
	requests int64
	// 上次请求完成时间，用于计算ewma值
	last int64
	// 上次选择的时间点
	pick int64
	// 节点名称
	name string
}

func (c *Conn) healthy() bool {
	return atomic.LoadUint64(&c.success) > throttleSuccess
}

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
		return penalty
	}
	return load
}
