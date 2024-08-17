package v5

import (
	"context"
	"emicro/v5/registry"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"time"
)

var (
	_ resolver.Builder  = (*grpcResolverBuilder)(nil)
	_ resolver.Resolver = (*grpcResolver)(nil)
)

type grpcResolverBuilder struct {
	registry registry.Registry
	timeout  time.Duration
}

func (b *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	res := &grpcResolver{
		cc:       cc,
		target:   target,
		timeout:  b.timeout,
		registry: b.registry,
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
func (b *grpcResolverBuilder) Scheme() string {
	return "registry"
}

func NewResolverBuilder(registry registry.Registry, timeout time.Duration) resolver.Builder {
	return &grpcResolverBuilder{registry: registry, timeout: timeout}
}

type grpcResolver struct {
	// - "dns://some_authority/foo.bar"
	//   Target{Scheme: "dns", Authority: "some_authority", Endpoint: "foo.bar"}
	// registry:///localhost:8081
	// builder *resolver.Builder
	target   resolver.Target
	registry registry.Registry
	cc       resolver.ClientConn
	timeout  time.Duration
	close    chan struct{}
}

// ResolveNow 立刻解析——立刻执行服务发现——立刻去问一下注册中心
func (r *grpcResolver) ResolveNow(options resolver.ResolveNowOptions) {
	// 重新获取一下所有服务
	r.resolve()
}

func (r *grpcResolver) resolve() {
	serviceName := r.target.Endpoint
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	instances, err := r.registry.ListServices(ctx, serviceName)
	if err != nil {
		r.cc.ReportError(err)
		return
	}
	address := make([]resolver.Address, 0, len(instances))
	for _, si := range instances {
		address = append(address, newAddress(si))
	}
	err = r.cc.UpdateState(resolver.State{Addresses: address})
	if err != nil {
		r.cc.ReportError(err)
		return
	}
}

func newAddress(ins registry.ServiceInstance) resolver.Address {
	return resolver.Address{
		// 定位信息，ip+端口
		Addr:       ins.Address,
		ServerName: ins.Name,
		// 可能还有其它字段
		Attributes: attributes.New("weight", ins.Weight).
			WithValue("group", ins.Group),
	}
}

func (r *grpcResolver) watch() {
	events, err := r.registry.Subscribe(r.target.Endpoint)
	if err != nil {
		//return err
		r.cc.ReportError(err)
		return
	}
	//go func() {
	//	for {
	//		select {
	//		case <-events:
	//			r.resolve()
	//		case <-r.close:
	//			close(r.close)
	//			return
	//		}
	//	}
	//}()
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
		case <-r.close:
			close(r.close)
		}
	}
}

// Close closes the resolver.
func (r *grpcResolver) Close() {
	// 有一个隐含的假设，就是 grpc 只会调用这个方法一次
	r.close <- struct{}{}
	//close(r.close)
}
