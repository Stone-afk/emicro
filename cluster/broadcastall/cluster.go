package broadcastall

import (
	"context"
	"emicro/v5/registry"
	"fmt"
	"google.golang.org/grpc"
	"reflect"
	"sync"
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
		ok, ch := isBroadCast(ctx)
		if !ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		instances, err := b.registry.ListServices(ctx, b.service)
		if err != nil {
			ch <- Resp{Err: err}
			return nil
		}
		typ := reflect.TypeOf(reply).Elem()
		var wg sync.WaitGroup
		wg.Add(len(instances))
		for _, instance := range instances {
			in := instance
			go func() {
				insCC, er := grpc.Dial(in.Address, b.dialOptions...)
				if er != nil {
					ch <- Resp{Err: er}
					wg.Done()
					return
				}
				newReply := reflect.New(typ).Interface()
				err = invoker(ctx, method, req, newReply, insCC, opts...)
				//err = conn.Invoke(ctx, method, req, val, opts...)

				//// 这种写法的风险在于，如果用户没有接收响应，
				//// 那么这里会阻塞导致 goroutine 泄露
				//ch <- Resp{Err: err, Val: newReply}
				// 如果没有人接收，就会堵住
				select {
				case <-ctx.Done():
					err = fmt.Errorf("响应没有人接收, %w", ctx.Err())
				case ch <- Resp{Err: er, Val: newReply}:
				}
				wg.Done()
			}()
		}
		go func() {
			wg.Wait()
			// 要记得 close 掉，不然用户不知道还有没有数据
			// 用户在调用的时候是不知道有多少个实例还存活着
			close(ch)
		}()
		return err
	}
}

// Resp 没有办法用泛型
type Resp struct {
	Val any
	Err error
}

type key struct{}

func UsingBroadCast(ctx context.Context) (context.Context, <-chan Resp) {
	ch := make(chan Resp)
	return context.WithValue(ctx, key{}, ch), ch
}

func isBroadCast(ctx context.Context) (bool, chan Resp) {
	val := ctx.Value(key{})
	if val != nil {
		res, ok := val.(chan Resp)
		if ok {
			return ok, res
		}
	}
	return false, nil
}
