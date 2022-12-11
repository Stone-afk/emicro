package rpc

import (
	"context"
	"emicro/v4/rpc/message"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message.Request) (*message.Response, error)
}

type Service interface {
	ServiceName() string
}
