package roundrobin

import (
	"emicro/loadbalance"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"sync"
	"sync/atomic"
)

const WeightRoundRobin = "WEIGHT_ROUND_ROBIN"

type WeightPicker struct {
	conns  []*weightConn
	mutex  sync.Mutex
	filter loadbalance.Filter
}

func (p *WeightPicker) Pick(info loadbalance.PickInfo) (loadbalance.PickResult, error) {
	if len(p.conns) == 0 {
		return loadbalance.PickResult{}, loadbalance.ErrNoSubConnAvailable
	}
	var totalWeight uint32
	var maxWeightConn *weightConn
	p.mutex.Lock()
	for _, node := range p.conns {
		if !p.filter(info, node.address) {
			continue
		}
		totalWeight += node.efficientWeight
		node.currentWeight += node.efficientWeight
		if maxWeightConn == nil || maxWeightConn.currentWeight < node.currentWeight {
			maxWeightConn = node
		}
	}
	maxWeightConn.currentWeight -= totalWeight
	p.mutex.Unlock()
	return loadbalance.PickResult{
		SubConn: maxWeightConn.SubConn,
		Done: func(info loadbalance.DoneInfo) {
			for {
				// 这里就是一个棘手的地方了
				// 按照算法，如果调用没有问题，那么增加权重
				// 如果调用有问题，减少权重

				// 直接减是很危险的事情，因为你可能 0 - 1 直接就最大值了
				// 也就是说一个节点不断失败不断失败，最终反而权重最大
				// 类似地，如果一个节点不断加不断加，最大值加1反而变最小值
				// if info.Err != nil {
				// 	atomic.AddUint32(&res.weight, -1)
				// } else {
				// 	atomic.AddUint32(&res.weight, 1)
				// }
				// 所以可以考虑 CAS 来，或者在 weightConn 里面设置一个锁

				weight := atomic.LoadUint32(&(maxWeightConn.efficientWeight))
				if info.Err != nil && weight == 0 {
					return
				}
				if info.Err == nil && weight == math.MaxUint32 {
					return
				}
				newWeight := weight
				if info.Err == nil {
					newWeight += 1
				} else {
					newWeight -= 1
				}
				if atomic.CompareAndSwapUint32(&(maxWeightConn.efficientWeight), weight, newWeight) {
					return
				}
			}
		},
	}, nil
}

type WeightPickerBuilder struct {
	Filter loadbalance.Filter
}

func (b *WeightPickerBuilder) Build(info base.PickerBuildInfo) loadbalance.Picker {
	conns := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 这里你可以考虑容错，例如服务器没有配置权重，给一个默认的权重
		// 但是我认为这种容错会让用户不经意间出 BUG，所以我这里不会校验，而是直接让它 panic
		// 这是因为 gRPC 确实没有设计 error 作为返回值
		weight := conInfo.Address.Attributes.Value("weight").(uint32)
		conns = append(conns, &weightConn{
			SubConn:         con,
			weight:          weight,
			currentWeight:   weight,
			efficientWeight: weight,
			address:         conInfo.Address,
		})
	}
	filter := b.Filter
	if filter == nil {
		filter = func(info loadbalance.PickInfo, address resolver.Address) bool {
			return true
		}
	}
	return &WeightPicker{
		conns:  conns,
		filter: filter,
	}
}

func (b *WeightPickerBuilder) Name() string {
	return WeightRoundRobin
}

type weightConn struct {
	// Initial weight
	weight uint32
	// Current weight
	currentWeight uint32
	// Effective weight, we will dynamically adjust the weight in the whole process
	efficientWeight uint32
	loadbalance.SubConn
	address resolver.Address
}
