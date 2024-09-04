package broadcastall

import (
	"context"
	"emicro/v5/registry"
	"google.golang.org/grpc"
)

type ClusterBuilder struct {
	service     string
	registry    registry.Registry
	dialOptions []grpc.DialOption

	// 还可以考虑设计成这种样子，然后和注册中心解耦
	// 不过在一个框架内部，耦合也没啥关系
	//	findServes func(ctx) []ServiceInstance
}

func NewClusterBuilder(r registry.Registry, service string, dialOptions ...grpc.DialOption) *ClusterBuilder {
	return &ClusterBuilder{
		registry:    r,
		service:     service,
		dialOptions: dialOptions,
	}
}

func (b *ClusterBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	// method: users.UserService/GetByID
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		panic("")
	}
}
