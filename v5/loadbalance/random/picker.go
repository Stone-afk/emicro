package random

import (
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
)

const Random = "RANDOM"

var (
	_ balancer.Picker    = (*Picker)(nil)
	_ base.PickerBuilder = (*PickerBuilder)(nil)
)

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	connections := make([]conn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		connections = append(connections, conn{
			SubConn: con,
			address: conInfo.Address,
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
	return Random
}

type Picker struct {
	length      int
	connections []conn
	filter      loadbalance.Filter
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	candidates := make([]conn, 0, len(p.connections))
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

type conn struct {
	balancer.SubConn
	address resolver.Address
}
