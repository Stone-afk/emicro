package ratelimit

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"time"
)

// TokenBucketLimiter 基于令牌桶的限流
// 大多数时候我们不需要自己手写算法，直接使用
// golang.org/x/time/rate
// 这里我们还是会手写一个
type TokenBucketLimiter struct {
	tokens chan struct{}
	close  chan struct{}
}

// NewTokenBucketLimiter buffer 最多能缓存住多少 token
// interval 多久产生一个令牌
func NewTokenBucketLimiter(buffer int, interval time.Duration) *TokenBucketLimiter {
	tokens := make(chan struct{}, buffer)
	closeCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-closeCh:
				// 关闭
				return
			case <-ticker.C:
				select {
				//case <- res.close:
				// 关闭。在这里其实可以没有这个分支
				//return
				case tokens <- struct{}{}:
				default:
					// 加 default 分支防止一直没有人取令牌，我们这里不能正常退出
				}
			}
		}
		//select {
		//case <- ticker.C:
		//	res.tokens <- struct{}{}
		//case <- res.closed:
		//	close(res.tokens)
		//	return
		//}
		//// for range ticker.C {
		////
		//// 	res.tokens <- struct{}{}
		////
		//// 	// 这个地方你可能放满
		//// 	// select {
		//// 	// case res.tokens <- struct{}{}:
		//// 	// default:
		//// 	//
		//// 	// }
		//// }
	}()
	return &TokenBucketLimiter{
		tokens: tokens,
		close:  closeCh,
	}
}

//func (t TokenBucketLimiter) BuildUnary() grpc.UnaryServerInterceptor {
//	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
//
//		// select {
//		// case <- ctx.Done():
//		// 	// 缺陷是 channel 是 FIFO 的
//		// 	// 意味着等待最久的，会拿到令牌
//		// 	// 这意味着，你大概率在业务处理的时候会超时。要小心不同超时时间设置
//		// 	return ctx.Err()
//		// case _, ok := <- t.tokens:
//		// 	if ok {
//		// 		return invoker(ctx, method, req, reply, cc, opts...)
//		// 	}
//		// }
//
//		// 怎么样处理？
//		select {
//		case _, ok := <- t.tokens:
//			if ok {
//				return handler(ctx, req)
//			}
//		default:
//			// 拿不到令牌就直接拒绝
//		}
//
//		// 熔断限流降级之间区别在这里了
//		// 1. 返回默认值 get_user -> GetUserResp
//		// 2. 打个标记位，后面执行快路径，或者兜底路径
//		return nil, errors.New("你被限流了")
//
//	}
//}

func (l *TokenBucketLimiter) LimitUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-l.close:
			// 已经关闭了
			// 这里你可以决策，如果认为限流器被关了，就代表不用限流，那么就直接发起调用。
			// 这种情况下，还要考虑提供 Start 方法重启限流器
			// 我这里采用另外一种语义，就是我认为限流器被关了，其实代表的是整个应用关了，所以我这里退出
			return nil, errors.New("emicro: 系统未被保护")
		case _, ok := <-l.tokens:
			if ok {
				return handler(ctx, req)
			}
		default:
			// 拿不到令牌就直接限流拒绝
		}
		// 熔断限流降级之间区别在这里了
		// 1. 返回默认值 get_user -> GetUserResp
		// 2. 打个标记位，后面执行快路径，或者兜底路径
		return nil, errors.New("emicro: 被限流了")
	}
}

func (l *TokenBucketLimiter) Close() error {
	// 直接关闭就可以
	// 多次关闭的情况我们就不处理了，用户需要自己来保证
	close(l.close)
	return nil
}
