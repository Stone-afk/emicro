package roundrobin

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"sync"
)

const RoundRobin = "ROUND_ROBIN"

type Picker struct {
	cnt    uint64
	conns  []conn
	mutex  sync.Mutex
	filter loadbalance.Filter
}

func (p *Picker) Pick(info loadbalance.PickInfo) (loadbalance.PickResult, error) {
	//It is theoretically feasible to use atomic operations instead of locks,
	// But the final effect is not a strict polling, but a rough polling
	//In this case, why not use random directly?
	// cnt := atomic.AddUint32(&b.cnt, 1)
	// index := cnt % b.length
	// atomic.StoreUint32(&b.cnt, index)
	p.mutex.Lock()
	defer p.mutex.Unlock()
	candidates := make([]conn, 0, len(p.conns))
	for _, c := range p.conns {
		if !p.filter(info, c.address) {
			continue
		}
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		return loadbalance.PickResult{}, loadbalance.ErrNoSubConnAvailable
	}
	index := p.cnt % uint64(len(candidates))
	p.cnt += 1
	return loadbalance.PickResult{
		SubConn: candidates[index].SubConn,
		// Used to design a feedback load balancing strategy
		Done: func(info loadbalance.DoneInfo) {
			//It can be labeled as unhealthy
			// if info. Err != nil {
			// }

			// This place is a magical place
			// The effect is to adjust your load balancing strategy according to the call result
			// If something goes wrong
			// if info. Err != nil {
			//Try to make the subConn unavailable or temporarily remove it
			// }

			// 实际上，这里你是要考虑如果调用失败，
			// 会不会是客户端和服务端的网络不通，
			// 按照道理来说，是需要将这个连不通的节点删除的
			// 但是删除之后又要考虑一段时间之后加回来
		},
	}, nil
}

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) loadbalance.Picker {
	conns := make([]conn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		conns = append(conns, conn{
			SubConn: con,
			address: conInfo.Address,
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
	return RoundRobin
}

type conn struct {
	loadbalance.SubConn
	address resolver.Address
}
