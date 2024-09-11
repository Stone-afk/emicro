package ratelimit

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"time"
)

var _ ServerLimiter = (*LeakyBucketLimiter)(nil)

type LeakyBucketLimiter struct {
	close    chan struct{}
	producer *time.Ticker
}

func NewLeakyBucketLimiter(interval time.Duration) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		close:    make(chan struct{}),
		producer: time.NewTicker(interval),
	}
}

func (l *LeakyBucketLimiter) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		select {
		case <-ctx.Done():
			// 等令牌过期了
			// 这里你也可以考虑回调拒绝策略
			return nil, ctx.Err()
		case <-l.close:
			// 已经关闭了
			// 这里你可以决策，如果认为限流器被关了，就代表不用限流，那么就直接发起调用。
			// 这种情况下，还要考虑提供 Start 方法重启限流器
			// 我这里采用另外一种语义，就是我认为限流器被关了，其实代表的是整个应用关了，所以我这里退出
			return nil, errors.New("emicro: 系统未被保护")
		case <-l.producer.C:
			return handler(ctx, req)
		}
	}
}

func (l *LeakyBucketLimiter) Close() error {
	// 直接关闭就可以
	// 多次关闭的情况我们就不处理了，用户需要自己来保证
	close(l.close)
	l.producer.Stop()
	return nil
}
