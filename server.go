package emicro

import (
	"context"
	"emicro/registry"
	"google.golang.org/grpc"
	"net"
	"time"
)

type ServerOption func(server *Server)

type Server struct {
	*grpc.Server
	registry registry.Registry
	listener net.Listener
	// 单个操作的超时时间，一般用于和注册中心打交道
	timeout time.Duration
	name    string
	weight  uint32
	group   string
}

func NewServer(name string, opts ...ServerOption) *Server {
	res := &Server{
		name:   name,
		Server: grpc.NewServer(),
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func ServerWithGroup(group string) ServerOption {
	return func(server *Server) {
		server.group = group
	}
}

func ServerWithWeight(weight uint32) ServerOption {
	return func(server *Server) {
		server.weight = weight
	}
}

func ServerWithRegistry(r registry.Registry) ServerOption {
	return func(server *Server) {
		server.registry = r
	}
}

func ServerWithTimeout(timeout time.Duration) ServerOption {
	return func(server *Server) {
		server.timeout = timeout
	}
}

func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	// 这边开始注册
	// 一定是先启动端口再注册
	// 严格地来说，是服务都启动了，才注册
	// 用户决定使用注册中心
	if s.registry != nil {
		// defer s.r.Unregister()
		ctx, cacel := context.WithTimeout(context.Background(), s.timeout)
		si := registry.ServiceInstance{
			Name:    s.name,
			Group:   s.group,
			Weight:  s.weight,
			Address: listener.Addr().String(),
		}
		// 要确保端口启动之后才能注册
		err = s.registry.Register(ctx, si)
		cacel()
		if err != nil {
			return err
		}
		// defer func() {
		// 	s.registry.UnRegister(ctx, s.si)
		// }()
	}
	return s.Serve(listener)
}

func (s *Server) Close() error {
	// 可以在这里 Unregister
	// s.r.Unregister()
	// 这里可以插入你的优雅退出逻辑
	// s.listener.Close()
	return nil
}
