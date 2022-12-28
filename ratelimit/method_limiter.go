package ratelimit

import (
	"context"
	"google.golang.org/grpc"
)

type MethodLimiter struct {
	Limiter
	FullMethod string
}

//// NewMethodLimiter 隔多久产生一个令牌
//func NewMethodLimiter(fullMethod string, limiter Limiter) *MethodLimiter {
//	return &MethodLimiter{
//		Limiter: limiter,
//		FullMethod: fullMethod,
//	}
//}

func (m *MethodLimiter) LimitUnary() grpc.UnaryServerInterceptor {
	interceptor := m.Limiter.LimitUnary()
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if info.FullMethod == m.FullMethod {
			return interceptor(ctx, req, info, handler)
		}
		return handler(ctx, req)
	}
}
