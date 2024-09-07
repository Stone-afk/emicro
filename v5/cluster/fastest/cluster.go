package fastest

import (
	"context"
	"emicro/v5/registry"
	"google.golang.org/grpc"
	"reflect"
	"sync"
)

type ClusterBuilder struct {
	service     string
	registry    registry.Registry
	dialOptions []grpc.DialOption
}

func NewClusterBuilder(r registry.Registry, service string, dialOptions ...grpc.DialOption) *ClusterBuilder {
	return &ClusterBuilder{
		registry:    r,
		service:     service,
		dialOptions: dialOptions,
	}
}

//func (b *ClusterBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
//	return func(ctx context.Context, method string, req, reply interface{},
//		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
//		if !isBroadCast(ctx) {
//			return invoker(ctx, method, req, reply, cc, opts...)
//		}
//		ins, err := b.registry.ListServices(ctx, b.service)
//		if err != nil {
//			return err
//		}
//		typ := reflect.TypeOf(reply).Elem()
//		ch := make(chan resp)
//		for _, instance := range ins {
//			in := instance
//			go func() {
//				conn, er := grpc.Dial(in.Address, b.opts...)
//				if er != nil {
//					ch <- resp{err: er}
//					return
//				}
//				r := reflect.New(typ)
//				val := r.Interface()
//				err = conn.Invoke(ctx, method, req, val, opts...)
//				select {
//				case ch <- resp{err: err, val: r}:
//				default:
//				}
//			}()
//		}
//		select {
//		case r := <-ch:
//			if r.err == nil {
//				reflect.ValueOf(reply).Elem().Set(r.val.Elem())
//			}
//			return r.err
//		case <-ctx.Done():
//			// 实际上这里是否监听 ctx 不重要，因为我们可以预期 grpc 会在超时的时候返回，走到上面的 error 分支
//			return ctx.Err()
//		}
//	}
//}

func (b *ClusterBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ok, ch := isBroadCast(ctx)
		if !ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		defer func() {
			close(ch)
		}()
		typ := reflect.TypeOf(reply).Elem()
		instances, err := b.registry.ListServices(ctx, b.service)
		if err != nil {
			return err
		}
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
				// 如果没有人接收，就会堵住
				select {
				case ch <- Resp{Err: er, Val: newReply}:
				default:
				}
				wg.Done()
			}()
		}
		wg.Wait()
		return err
	}
}

type key struct{}

func UsingBroadCast(ctx context.Context) (context.Context, <-chan Resp) {
	ch := make(chan Resp)
	return context.WithValue(ctx, key{}, ch), ch
}

func isBroadCast(ctx context.Context) (bool, chan Resp) {
	val := ctx.Value(key{})
	if val == nil {
		return false, nil
	}
	res, ok := val.(chan Resp)
	return ok, res
}

type Resp struct {
	Val any
	Err error
}
