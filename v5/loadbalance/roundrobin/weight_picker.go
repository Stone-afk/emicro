package roundrobin

import (
	"emicro/v5/loadbalance"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math"
	"sync"
)

const WeightRoundRobin = "WEIGHT_ROUND_ROBIN"

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
			SubConn:         con,
			weight:          weight,
			currentWeight:   weight,
			efficientWeight: weight,
			address:         conInfo.Address,
			name:            conInfo.Address.Addr,
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
	return WeightRoundRobin
}

type WeightPicker struct {
	connections []*weightConn
	mutex       sync.Mutex
	filter      loadbalance.Filter
}

func (p *WeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.connections) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	var totalWeight uint32
	var maxWeightConn *weightConn
	//p.mutex.Lock()
	for _, con := range p.connections {
		if !p.filter(info, con.address) {
			continue
		}
		con.mutex.Lock()
		totalWeight += con.efficientWeight
		con.currentWeight += con.efficientWeight
		if maxWeightConn == nil || maxWeightConn.currentWeight < con.currentWeight {
			maxWeightConn = con
		}
		con.mutex.Unlock()
	}
	if maxWeightConn == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	maxWeightConn.mutex.Lock()
	maxWeightConn.currentWeight -= totalWeight
	maxWeightConn.mutex.Unlock()
	//p.mutex.Unlock()
	return balancer.PickResult{
		SubConn: maxWeightConn,
		Done: func(info balancer.DoneInfo) {
			//for {
			//	// 这里就是一个棘手的地方了
			//	// 按照算法，如果调用没有问题，那么增加权重
			//	// 如果调用有问题，减少权重
			//
			//	// 直接减是很危险的事情，因为你可能 0 - 1 直接就最大值了
			//	// 也就是说一个节点不断失败不断失败，最终反而权重最大
			//	// 类似地，如果一个节点不断加不断加，最大值加1反而变最小值
			//	// if info.Err != nil {
			//	// 	atomic.AddUint32(&res.weight, -1)
			//	// } else {
			//	// 	atomic.AddUint32(&res.weight, 1)
			//	// }
			//	// 所以可以考虑 CAS 来，或者在 weightConn 里面设置一个锁
			//
			//	weight := atomic.LoadUint32(&(maxWeightConn.efficientWeight))
			//	if info.Err != nil && weight == 0 {
			//		return
			//	}
			//	if info.Err == nil && weight == math.MaxUint32 {
			//		return
			//	}
			//	newWeight := weight
			//	if info.Err == nil {
			//		newWeight += 1
			//	} else {
			//		newWeight -= 1
			//	}
			//	if atomic.CompareAndSwapUint32(&(maxWeightConn.efficientWeight), weight, newWeight) {
			//		return
			//	}
			//}
			maxWeightConn.mutex.Lock()
			defer maxWeightConn.mutex.Unlock()
			if info.Err != nil && maxWeightConn.weight == 0 {
				return
			}
			if info.Err == nil && maxWeightConn.efficientWeight == math.MaxUint32 {
				return
			}
			if info.Err != nil {
				maxWeightConn.efficientWeight--
			} else {
				maxWeightConn.efficientWeight++
			}
		},
	}, nil
}

type weightConn struct {
	name string
	// Initial weight
	weight uint32
	// Current weight
	currentWeight uint32
	// Effective weight, we will dynamically adjust the weight in the whole process
	efficientWeight uint32
	available       bool
	balancer.SubConn
	mutex   sync.Mutex
	address resolver.Address
}
