package ratelimit

import (
	"context"
	"google.golang.org/grpc"
)

type ServerMethodLimiter struct {
	ServerLimiter
	FullMethod string
}

// NewServerMethodLimiter 隔多久产生一个令牌
func NewServerMethodLimiter(fullMethod string, limiter ServerLimiter) *ServerMethodLimiter {
	return &ServerMethodLimiter{
		ServerLimiter: limiter,
		FullMethod:    fullMethod,
	}
}

func (l *ServerMethodLimiter) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	interceptor := l.ServerLimiter.BuildServerInterceptor()
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if info.FullMethod == l.FullMethod {
			return interceptor(ctx, req, info, handler)
		}
		return handler(ctx, req)
	}
}
