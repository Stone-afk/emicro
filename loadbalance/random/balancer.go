package random

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
)

const Random = "RANDOM"

type Picker struct {
	conns  []conn
	filter loadbalance.Filter
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	candidates := make([]conn, 0, len(p.conns))
	for _, c := range p.conns {
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

type PickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	conns := make([]conn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		conns = append(conns, conn{
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
		conns:  conns,
		filter: filter,
	}
}

func (b *PickerBuilder) Name() string {
	return Random
}

type conn struct {
	balancer.SubConn
	address resolver.Address
}
