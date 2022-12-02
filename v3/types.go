//go:build v3

package emicro

import (
	"context"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message.Request) (*message.Response, error)
}

type Service interface {
	ServiceName() string
}
