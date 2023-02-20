package leastactive

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"sync"
	"sync/atomic"
)

const LeastActive = "LEAST_ACTIVE"

type Picker struct {
	cnt    uint64
	conns  []*conn
	mutex  sync.Mutex
	filter loadbalance.Filter
}

func (p *Picker) Pick(info loadbalance.PickInfo) (loadbalance.PickResult, error) {
	if len(p.conns) == 0 {
		return loadbalance.PickResult{}, loadbalance.ErrNoSubConnAvailable
	}
	// The disadvantage of using atomic operations is that they are not accurate enough
	// If the lock is used instead, the performance is too poor
	// Think about why?
	//p.mutex.Lock()
	//defer p.mutex.Unlock()

	var leastActive uint32 = math.MaxUint32
	var res *conn
	for _, con := range p.conns {
		if !p.filter(info, con.address) {
			continue
		}
		active := atomic.LoadUint32(&con.active)
		if active < leastActive {
			leastActive = active
			res = con
		}
	}
	atomic.AddUint32(&res.active, 1)
	return loadbalance.PickResult{
		SubConn: res.SubConn,
		Done: func(info loadbalance.DoneInfo) {
			atomic.AddUint32(&res.active, -1)
		},
	}, nil
}

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) loadbalance.Picker {
	conns := make([]*conn, 0, len(info.ReadySCs))
	for con, val := range info.ReadySCs {
		conns = append(conns, &conn{
			SubConn: con,
			address: val.Address,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info loadbalance.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &Picker{
		conns:  conns,
		filter: filter,
	}
}

func (b *PickerBuilder) Name() string {
	return LeastActive
}

type conn struct {
	active uint32
	loadbalance.SubConn
	address resolver.Address
}
