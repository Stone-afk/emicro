package roundrobin

import (
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"sync"
)

const RoundRobin = "ROUND_ROBIN"

var (
	_ balancer.Picker    = (*Picker)(nil)
	_ base.PickerBuilder = (*PickerBuilder)(nil)
)

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		connections = append(connections, conn{
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

func (b *PickerBuilder) Name() string {
	return RoundRobin
}

type Picker struct {
	cnt         uint64
	connections []balancer.SubConn
	mutex       sync.Mutex
	filter      loadbalance.Filter
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	//It is theoretically feasible to use atomic operations instead of locks,
	// But the final effect is not a strict polling, but a rough polling
	//In this case, why not use random directly?
	// cnt := atomic.AddUint32(&b.cnt, 1)
	// index := cnt % b.length
	// atomic.StoreUint32(&b.cnt, index)
	p.mutex.Lock()
	defer p.mutex.Unlock()
	candidates := make([]balancer.SubConn, 0, len(p.connections))
	for _, con := range p.connections {
		if !p.filter(info, con.(conn).address) {
			continue
		}
		candidates = append(candidates, con)
	}
	if len(candidates) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	index := p.cnt % uint64(len(candidates))
	p.cnt += 1
	return balancer.PickResult{
		SubConn: candidates[index],
		//SubConn: candidates[index].SubConn,
		// Used to design a feedback load balancing strategy
		Done: func(info balancer.DoneInfo) {
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

type conn struct {
	name string
	balancer.SubConn
	address resolver.Address
}
