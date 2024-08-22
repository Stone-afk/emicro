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
	//TODO implement me
	panic("implement me")
}

func (b *PickerBuilder) Name() string {
	return RoundRobin
}

type Picker struct {
	cnt    uint64
	connes []conn
	mutex  sync.Mutex
	filter loadbalance.Filter
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	//TODO implement me
	panic("implement me")
}

type conn struct {
	balancer.SubConn
	address resolver.Address
}
