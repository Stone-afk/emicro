package v5

import (
	"context"
	"emicro/v5/registry"
	"google.golang.org/grpc"
	"net"
	"time"
)

type ServerOption func(server *Server)

func ServerWithWeight(weight uint32) ServerOption {
	return func(server *Server) {
		server.weight = weight
	}
}

func ServerWithGroup(group string) ServerOption {
	return func(server *Server) {
		server.group = group
	}
}

func ServerWithRegistry(r registry.Registry) ServerOption {
	return func(server *Server) {
		server.registry = r
	}
}

type Server struct {
	name   string
	weight uint32
	group  string

	*grpc.Server
	listener net.Listener
	registry registry.Registry
	// 单个操作的超时时间，一般用于和注册中心打交道
	registerTimeout time.Duration
	si              registry.ServiceInstance
}

// Start 当用户调用这个方法的时候，就是服务已经准备好
func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener // 这边开始注册
	// 一定是先启动端口再注册
	// 严格地来说，是服务都启动了，才注册
	// 用户决定使用注册中心
	if s.registry != nil {
		// defer s.r.Unregister()
		ctx, cancel := context.WithTimeout(context.Background(), s.registerTimeout)
		defer cancel()
		s.si = registry.ServiceInstance{
			Name:    s.name,
			Group:   s.group,
			Weight:  s.weight,
			Address: listener.Addr().String(),
		}
		// 要确保端口启动之后才能注册
		err = s.registry.Register(ctx, s.si)
		if err != nil {
			return err
		}
		// 这里已经注册成功了
		//defer func() {
		// 忽略或者 log 一下错误
		// _ = s.registry.Close()
		// _ = s.registry.UnRegister(registry.ServiceInstance{})
		//}()
	}
	return s.Serve(listener)
}

func (s *Server) Close() error {
	// 可以在这里 Unregister
	// s.registry.Unregister()
	// 这里可以插入你的优雅退出逻辑
	// s.listener.Close()
	err := s.registry.Unregister(context.Background(), s.si)
	if err != nil {
		return err
	}
	s.GracefulStop()
	return nil
	// 这里可以插入优雅退出逻辑
	//return s.listener.Close()
}

func NewServer(name string, opts ...ServerOption) (*Server, error) {
	res := &Server{
		name:            name,
		Server:          grpc.NewServer(),
		registerTimeout: time.Second * 10,
	}
	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}