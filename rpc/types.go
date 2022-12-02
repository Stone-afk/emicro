package rpc

import (
	"context"
	"emicro/rpc/message"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message.Request) (*message.Response, error)
}

type Service interface {
	ServiceName() string
}
