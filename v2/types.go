//go:build v2

package emicro

import (
	"context"
	"emicro/v2/message"
)

type Proxy interface {
	Invoke(ctx context.Context, request *message.Request) (*message.Response, error)
}

type Service interface {
	ServiceName() string
}
