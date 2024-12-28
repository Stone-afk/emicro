package ratelimit

import (
	"context"
	_ "embed"
	"github.com/go-redis/redis/v9"
	"google.golang.org/grpc"
	"time"
)

//go:embed lua/slide_window.lua
var luaSlideWindow string

type RedisSlideWindowLimiter struct {
	key string
	// 窗口内的流量阈值
	maxRate int
	// 窗口大小，毫秒
	interval int64
	onReject rejectStrategy
	client   redis.Cmdable
}

func NewRedisSlideWindowLimiter(client redis.Cmdable, key string, maxRate int, interval time.Duration) *RedisSlideWindowLimiter {
	return &RedisSlideWindowLimiter{
		client:   client,
		key:      key,
		maxRate:  maxRate,
		interval: interval.Milliseconds(),
		onReject: defaultRejection,
	}
}

func (l *RedisSlideWindowLimiter) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		// 预期 lua 脚本会返回一个 bool 值，告诉我要不要限流
		// 使用 FullMethod，那就是单一方法上限流，比如说 GetById
		// 使用服务名来限流，那就是在单一服务上 users.UserService
		// 使用应用名，user-service
		limit, err := l.limit(ctx)
		if err != nil {
			// 正常来说，遇到 error 表示你也不知道要不要限流
			// 那么你可以选择限流，也可以选择不限流
			return nil, err
		}
		if limit {
			return l.onReject(ctx, req, info, handler)
		}
		return handler(ctx, req)
	}
}

func (l *RedisSlideWindowLimiter) limit(ctx context.Context) (bool, error) {
	now := time.Now()
	return l.client.Eval(ctx, luaSlideWindow, []string{l.key}, l.maxRate, l.interval, now.UnixMilli()).Bool()
}
