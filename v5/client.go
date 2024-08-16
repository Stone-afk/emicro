package v5

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

type ClientOption func(client *Client)

func ClientInsecure() ClientOption {
	return func(c *Client) {
		c.insecure = true
	}
}

type Client struct {
	insecure bool
	rb       resolver.Builder
}

func NewClient(opts ...ClientOption) *Client {
	res := &Client{}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (c *Client) Dial(ctx context.Context, service string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	panic("")
}
