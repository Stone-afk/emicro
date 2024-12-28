package random

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
)

const WeightRandom = "WEIGHT_RANDOM"

var (
	_ balancer.Picker    = (*WeightPicker)(nil)
	_ base.PickerBuilder = (*WeightPickerBuilder)(nil)
)

type WeightPickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *WeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 这里你可以考虑容错，例如服务器没有配置权重，给一个默认的权重
		// 但是我认为这种容错会让用户不经意间出 BUG，所以我这里不会校验，而是直接让它 panic
		// 这是因为 gRPC 确实没有设计 error 作为返回值
		weight := conInfo.Address.Attributes.Value("weight").(uint32)
		connections = append(connections, &weightConn{
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
		connections: connections,
		filter:      filter,
	}
}

func (b *WeightPickerBuilder) Name() string {
	return WeightRandom
}

type WeightPicker struct {
	length      int
	connections []*weightConn
	filter      loadbalance.Filter
}

func (p *WeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	var totalWeight uint32
	for _, con := range p.connections {
		if !p.filter(info, con.address) {
			continue
		}
		totalWeight += con.weight
	}
	val := rand.Intn(int(totalWeight) + 1)
	for _, con := range p.connections {
		if !p.filter(info, con.address) {
			continue
		}
		val = val - int(con.weight)
		if val <= 0 {
			return balancer.PickResult{
				SubConn: con,
				Done: func(info balancer.DoneInfo) {
					// 可以在这里修改权重，但是要考虑并发安全
					//In fact, here we can also consider adjusting the weight according to the call result,
					//Similar to that in roubin
				},
			}, nil
		}
	}
	// In fact, it is impossible to run here, because we must be able to find a value before
	return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
}

type weightConn struct {
	name string
	// Initial weight
	weight uint32
	balancer.SubConn
	address resolver.Address
}
