package ratelimit

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Limiter interface {
	LimitUnary() grpc.UnaryServerInterceptor
}

type rejectStrategy func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)

var defaultRejection rejectStrategy = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return nil, status.Errorf(codes.ResourceExhausted, "触发限流 %s", info.FullMethod)
}

var markLimitedRejection rejectStrategy = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx = context.WithValue(ctx, "limited", true)
	return handler(ctx, req)
}

type Guardian interface {
	Allow(ctx context.Context, req interface{}) (cb func(), err error)
	AllowV1(ctx context.Context, req interface{}) (cb func(), resp interface{}, err error)
	OnRejection(ctx context.Context, req interface{}) (interface{}, error)
}
