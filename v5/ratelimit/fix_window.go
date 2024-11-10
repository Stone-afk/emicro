package ratelimit

import (
	"context"
	"google.golang.org/grpc"
	"sync/atomic"
	"time"
)

var _ ServerLimiter = (*FixWindowLimiter)(nil)

type FixWindowLimiter struct {
	// 窗口大小
	interval int64
	// 在 interval 内最多允许 rate 个请求
	maxRate  int64
	cnt      int64
	onReject rejectStrategy
	// 窗口的起始时间
	latestWindowStartTimestamp int64
	//mutex sync.Mutex

	//onReject rejectStrategy
}

// NewFixWindowLimiter
// interval => 窗口多大
// max 这个窗口内，能够执行多少个请求
func NewFixWindowLimiter(interval time.Duration, maxRate int64) *FixWindowLimiter {
	return &FixWindowLimiter{
		maxRate:  maxRate,
		onReject: defaultRejection,
		interval: interval.Nanoseconds(),
	}
}

func (l *FixWindowLimiter) OnReject(onReject rejectStrategy) *FixWindowLimiter {
	l.onReject = onReject
	return l
}

func (l *FixWindowLimiter) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		current := int64(time.Now().Nanosecond())
		windowStart := atomic.LoadInt64(&l.latestWindowStartTimestamp)
		//cnt := atomic.LoadInt64(&l.cnt)
		// 如果要是最近窗口的起始时间 + 窗口大小 < 当前时间戳
		// 说明换窗口了
		if windowStart+l.interval < current {
			// 换窗口了
			// 重置了 latestWindowStartTimestamp
			// 重置，注意，这里任何一步 CAS 操作失败，都意味着有别的 goroutine 重置了
			// 所以我们失败了就直接忽略
			if atomic.CompareAndSwapInt64(&l.latestWindowStartTimestamp, windowStart, current) {
				atomic.StoreInt64(&l.cnt, 0)
			}
		}
		// 检查这个窗口还能不能处理新请求
		// 先取号
		cnt := atomic.AddInt64(&l.cnt, 1)
		// 超过上限了
		if cnt > l.maxRate {
			return l.onReject(ctx, req, info, handler)
		}
		return handler(ctx, req)
	}
}
