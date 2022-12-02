//go:build v2

package emicro

import "context"

type onewayKey struct{}

func CtxWithOneway(ctx context.Context) context.Context {
	return context.WithValue(ctx, onewayKey{}, true)
}

func isOneway(ctx context.Context) bool {
	return ctx.Value(onewayKey{}) != nil
}
