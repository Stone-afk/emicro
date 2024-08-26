package hash

import (
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

const ConsistentHash = "CONSISTENT_HASH"

var (
	_ balancer.Picker    = (*ConsistentPicker)(nil)
	_ base.PickerBuilder = (*ConsistentPickerBuilder)(nil)
)

type ConsistentPicker struct {
	length      int
	filter      loadbalance.Filter
	connections []balancer.SubConn
}

func (b *ConsistentPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if b.length == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// 在这个地方你拿不到请求，无法做根据请求特性做负载均衡
	//idx := info.Ctx.Value("user_id")
	//idx := info.Ctx.Value("hash_code")

	return balancer.PickResult{
		SubConn: b.connections[0],
		Done: func(info balancer.DoneInfo) {

		},
	}, nil
}

type ConsistentPickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *ConsistentPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for c := range info.ReadySCs {
		connections = append(connections, c)
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info balancer.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &ConsistentPicker{
		filter:      filter,
		connections: connections,
		length:      len(connections),
	}
}

func (b *ConsistentPickerBuilder) Name() string {
	return ConsistentHash
}
