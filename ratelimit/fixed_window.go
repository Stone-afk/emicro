package ratelimit

import (
	"context"
	"google.golang.org/grpc"
	"sync/atomic"
	"time"
)

type FixWindowLimiter struct {
	interval int64
	// 在 interval 内最多允许 rate 个请求
	maxRate                    int64
	cnt                        int64
	onReject                   rejectStrategy
	latestWindowStartTimestamp int64
}

// NewFixWindowLimiter
// interval => 窗口多大
// max 这个窗口内，能够执行多少个请求
func NewFixWindowLimiter(interval time.Duration, maxRate int64) *FixWindowLimiter {
	return &FixWindowLimiter{
		interval: interval.Nanoseconds(),
		maxRate:  maxRate,
		onReject: defaultRejection,
	}
}

func (t *FixWindowLimiter) OnReject(onReject rejectStrategy) *FixWindowLimiter {
	t.onReject = onReject
	return t
}

func (t *FixWindowLimiter) LimitUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		current := time.Now().Nanosecond()
		window := atomic.LoadInt64(&t.latestWindowStartTimestamp)
		// 如果要是最近窗口的起始时间 + 窗口大小 < 当前时间戳
		// 说明换窗口了
		if window+t.interval < int64(current) {
			// 换窗口了
			// 重置了 latestWindowStartTimestamp
			// 重置，注意，这里任何一步 CAS 操作失败，都意味着有别的 goroutine 重置了
			// 所以我们失败了就直接忽略
			if atomic.CompareAndSwapInt64(&t.latestWindowStartTimestamp, window, 0) {
				atomic.StoreInt64(&t.cnt, 0)
			}
		}
		// 检查这个窗口还能不能处理新请求
		// 先取号
		cnt := atomic.AddInt64(&t.cnt, 1)
		// 超过上限了
		if cnt > t.maxRate {
			return t.onReject(ctx, req, info, handler)
		}
		return handler(ctx, req)
	}
}
