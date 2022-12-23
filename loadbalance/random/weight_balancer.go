package random

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
	"sync"
)

const WeightRandom = "WEIGHT_RANDOM"

type WeightPicker struct {
	conns  []*weightConn
	mutex  sync.Mutex
	filter loadbalance.Filter
}

func (p *WeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	var totalWeight uint32
	for _, node := range p.conns {
		if !p.filter(info, node.address) {
			continue
		}
		totalWeight += node.weight
	}
	val := rand.Intn(int(totalWeight))
	for _, con := range p.conns {
		if !p.filter(info, con.address) {
			continue
		}
		val = val - int(con.weight)
		if val <= 0 {
			return balancer.PickResult{
				SubConn: con.SubConn,
				Done: func(info balancer.DoneInfo) {
					//In fact, here we can also consider adjusting the weight according to the call result,
					//Similar to that in roubin
				},
			}, nil
		}
	}
	// In fact, it is impossible to run here, because we must be able to find a value before
	return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
}

type WeightPickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *WeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	conns := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 这里你可以考虑容错，例如服务器没有配置权重，给一个默认的权重
		// 但是我认为这种容错会让用户不经意间出 BUG，所以我这里不会校验，而是直接让它 panic
		// 这是因为 gRPC 确实没有设计 error 作为返回值
		weight := conInfo.Address.Attributes.Value("weight").(uint32)
		conns = append(conns, &weightConn{
			SubConn: con,
			weight:  weight,
			address: conInfo.Address,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &WeightPicker{
		conns:  conns,
		filter: filter,
	}
}

func (b *WeightPickerBuilder) Name() string {
	return WeightRandom
}

type weightConn struct {
	// Initial weight
	weight uint32
	balancer.SubConn
	address resolver.Address
}
