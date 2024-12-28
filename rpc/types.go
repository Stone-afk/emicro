package rpc

import (
	"context"
	message2 "emicro/rpc/message"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message2.Request) (*message2.Response, error)
}

type Service interface {
	Name() string
}
