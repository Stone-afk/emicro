package emicro

import (
	"context"
	"emicro/registry"
	"google.golang.org/grpc/resolver"
	"log"
	"time"
)

type grpcResolverBuilder struct {
	r       registry.Registry
	timeout time.Duration
}

func NewResolverBuilder(r registry.Registry, timeout time.Duration) resolver.Builder {
	return &grpcResolverBuilder{
		r:       r,
		timeout: timeout,
	}
}

func (r *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	res := &grpcResolver{
		target:   target,
		cc:       cc,
		registry: r.r,
		close:    make(chan struct{}, 1),
		timeout:  r.timeout,
	}
	res.resolve()
	go res.watch()
	return res, nil
}

// 伪代码
// func (r *grpcResolverBuilder) Build1(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
// 	res := &grpcResolver{}
// 	// 要在这里初始化连接，然后更新 cc 里面的连接信息
// 	address := make([]resolver.Address, 10)
// 	cc.UpdateState(resolver.State{
// 		Addresses: address,
// 	})
// 	return res, nil
// }

// Scheme 返回一个固定的值，registry 代表的是我们设计的注册中心
func (r *grpcResolverBuilder) Scheme() string {
	return "registry"
}

type grpcResolver struct {
	target   resolver.Target
	cc       resolver.ClientConn
	registry registry.Registry
	timeout  time.Duration
	close    chan struct{}
	// builder *resolver.Builder
}

// ResolveNow 立刻解析——立刻执行服务发现——立刻去问一下注册中心
func (r *grpcResolver) ResolveNow(options resolver.ResolveNowOptions) {
	// 重新获取一下所有服务
	r.resolve()
}

func (r *grpcResolver) resolve() {
	serviceName := r.target.Endpoint
	// 这个就是可用服务实例（节点）列表
	// 你要考虑设置超时
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	instances, err := r.registry.ListServices(ctx, serviceName)
	cancel()
	if err != nil {
		r.cc.ReportError(err)
	}
	address := make([]resolver.Address, 0, len(instances))
	for _, ins := range instances {
		address = append(address, resolver.Address{
			// 定位信息，ip+端口
			Addr: ins.Address,
			// 可能还有其它字段
			ServerName: ins.Name,
		})
	}
	err = r.cc.UpdateState(resolver.State{
		Addresses: address,
	})
	if err != nil {
		r.cc.ReportError(err)
	}
}

func (r *grpcResolver) watch() {
	events := r.registry.Subscribe(r.target.Endpoint)
	for {
		select {
		case <-events:
			// 做法一：立刻更新可用节点列表
			// 这种是幂等的

			// 在这里引入重试的机制
			r.resolve()
			// 做法二：精细化做法，非常依赖于事件顺序
			// 你这里收到的事件的顺序，要和在注册中心上发生的顺序一样
			// 少访问一次注册中心
			// switch event.Type {
			// case registry.EventTypeAdd:
			// 	state.Addresses = append(state.Addresses, resolver.Address{
			// 	Addr: event.Instance.Address,
			// 	})
			// 	cc.UpdateState(state)
			// 	// cc.AddAddress
			// case registry.EventTypeDelete:
			// 	event.Instance // 这是被删除的节点
			// case registry.EventTypeUpdate:
			// 	event.Instance // 这是被更新的，而且是更新后的节点
			//
			// }
			log.Println(events)
		case <-r.close:
			close(r.close)
			return
		}
	}
}

// Close closes the resolver.
func (r *grpcResolver) Close() {
	// 有一个隐含的假设，就是 grpc 只会调用这个方法一次
	// r.close <- struct{}{}

	// close(r.close)
	r.close <- struct{}{}
}
