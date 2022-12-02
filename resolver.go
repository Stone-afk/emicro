package emicro

import (
	"context"
	"emicro/registry"
	"google.golang.org/grpc/resolver"
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
}

// ResolveNow 立刻解析——立刻执行服务发现——立刻去问一下注册中心
func (r *grpcResolver) ResolveNow(options resolver.ResolveNowOptions) {
	// 重新获取一下所有服务
	r.resolve()
}

func (r *grpcResolver) resolve() {
	serviceName := r.target.Endpoint
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
			// 一种做法就是我们这边区别处理不同事件类型，然后更新数据
			// switch event.Type {
			//
			//			}
			// 另外一种做法就是我们这里采用的，每次事件发生的时候，就直接刷新整个可用服务列表
			r.resolve()
		case <-r.close:
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
