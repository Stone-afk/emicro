package leastactive

import (
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"sync"
	"sync/atomic"
)

const LeastActive = "LEAST_ACTIVE"

var (
	_ balancer.Picker    = (*Picker)(nil)
	_ base.PickerBuilder = (*PickerBuilder)(nil)
)

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]*conn, 0, len(info.ReadySCs))
	for con, val := range info.ReadySCs {
		connections = append(connections, &conn{
			SubConn: con,
			address: val.Address,
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

func (b *PickerBuilder) Name() string {
	return LeastActive
}

type Picker struct {
	cnt         uint64
	mutex       sync.Mutex
	filter      loadbalance.Filter
	connections []*conn
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// The disadvantage of using atomic operations is that they are not accurate enough
	// If the lock is used instead, the performance is too poor
	// Think about why?
	//p.mutex.Lock()
	//defer p.mutex.Unlock()
	var leastActive uint32 = math.MaxUint32
	var res *conn
	for _, con := range p.connections {
		if !p.filter(info, con.address) {
			continue
		}
		active := atomic.LoadUint32(&con.active)
		if active < leastActive {
			leastActive = active
			res = con
		}
	}
	if res == nil {
		// 你也可以考虑筛选完之后，没有任何符合条件的节点，就用默认节点
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	atomic.AddUint32(&res.active, 1)
	return balancer.PickResult{
		SubConn: res,
		Done: func(info balancer.DoneInfo) {
			atomic.AddUint32(&res.active, -1)
		},
	}, nil
}

type conn struct {
	name   string
	active uint32
	balancer.SubConn
	address resolver.Address
}
