package v5

import (
	"google.golang.org/grpc/resolver"
)

type ClientOption func(client *Client)

type Client struct {
	insecure bool
	rb       resolver.Builder
}
