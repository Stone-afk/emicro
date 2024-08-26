package random

import (
	"emicro/v5/loadbalance"
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
	candidates := make([]*weightConn, 0, len(p.connections))
	for _, c := range p.connections {
		if !p.filter(info, c.address) {
			continue
		}
		candidates = append(candidates, c)
	}
	index := rand.Intn(len(candidates))
	return balancer.PickResult{
		SubConn: candidates[index].SubConn,
	}, nil
}

type weightConn struct {
	name string
	// Initial weight
	weight uint32
	balancer.SubConn
	address resolver.Address
}
