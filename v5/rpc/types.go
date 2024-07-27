package rpc

import (
	"context"
	"emicro/v5/rpc/message"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message.Request) (*message.Response, error)
}

type Service interface {
	Name() string
}
