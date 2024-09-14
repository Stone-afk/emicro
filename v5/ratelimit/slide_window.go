package ratelimit

import (
	"container/list"
	"context"
	"google.golang.org/grpc"
	"sync"
	"time"
)

var _ ServerLimiter = (*SlideWindowLimiter)(nil)

type SlideWindowLimiter struct {
	// 上限
	maxRate int
	// 你需要一个 queue 来缓存住你窗口内每一个请求的时间戳
	queue    *list.List
	mutex    sync.RWMutex
	interval time.Duration
	onReject rejectStrategy
}

func (l *SlideWindowLimiter) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// info.FullMethod = "/user" 就限流
		// user 的和 order 的分开限流

		// user, ok :=req.(*GetUserReq)
		// if ok

		// 假如说现在是 3:17 interval 是一分钟
		current := time.Now()
		l.mutex.Lock()
		cnt := l.queue.Len()
		if cnt < l.maxRate {
			// 记住了请求的时间戳
			l.queue.PushBack(current)
			l.mutex.Unlock()
			return handler(ctx, req)
		}
		// 慢路径
		// 往前回溯（所以是减号），起始时间是 2:17
		windowStartTime := current.Add(-l.interval)

		// 假如说 reqTime 是 2:12，代表它其实已经不在这个窗口里面了
		reqTime := l.queue.Front()
		for reqTime != nil && reqTime.Value.(time.Time).Before(windowStartTime) {
			// 说明这个请求不在这个窗口范围内，移除窗口
			l.queue.Remove(reqTime)
			reqTime = l.queue.Front()
		}
		cnt = l.queue.Len()
		if cnt > l.maxRate {
			l.mutex.Unlock()
			return l.onReject(ctx, req, info, handler)
		}
		l.queue.PushBack(current)
		l.mutex.Unlock()
		return handler(ctx, req)
	}
}

func (l *SlideWindowLimiter) OnReject(onReject rejectStrategy) *SlideWindowLimiter {
	l.onReject = onReject
	return l
}

func NewSlideWindowLimiter(rate int, interval time.Duration) *SlideWindowLimiter {
	return &SlideWindowLimiter{
		maxRate:  rate,
		interval: interval,
		queue:    list.New(),
		onReject: defaultRejection,
	}
}
