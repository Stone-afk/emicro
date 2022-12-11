//go:build v4

package emicro

import (
	"context"
	"emicro/registry"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
	"time"
)

type ClientOption func(client *Client)

type Client struct {
	insecure bool
	rb       resolver.Builder
	balancer balancer.Builder
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Dial(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{grpc.WithResolvers(c.rb)}
	address := fmt.Sprintf("registry:///%s", serviceName)
	if c.insecure {
		opts = append(opts, grpc.WithInsecure())
	}
	if c.balancer != nil {
		opts = append(opts, grpc.WithDefaultServiceConfig(
			fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`,
				c.balancer.Name())))
	}
	return grpc.DialContext(ctx, address, opts...)
}

func ClientWithRegistry(r registry.Registry, timeout time.Duration) ClientOption {
	return func(client *Client) {
		client.rb = NewResolverBuilder(r, timeout)
	}
}

func ClientWithInsecure() ClientOption {
	return func(client *Client) {
		client.insecure = true
	}
}

// 伪代码
// func (c *Client) DialPsu(ctx context.Context, service string) (*grpc.ClientConn, error) {
// 	resolver := c.rb
//
// 	grpc.DialContext(ctx,
// 		"registry:///user-service",
// 		grpc.WithResolvers(resolver))
// }
