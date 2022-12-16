package emicro

import (
	"context"
	"emicro/registry"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"time"
)

type ClientOption func(client *Client)

type Client struct {
	insecure        bool
	rb              resolver.Builder
	balancerBuilder balancer.Builder
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
	if c.balancerBuilder != nil {
		opts = append(opts, grpc.WithDefaultServiceConfig(
			fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`,
				c.balancerBuilder.Name())))
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

func ClientWithPickerBuilder(name string, pickerBuilder base.PickerBuilder) ClientOption {
	return func(client *Client) {
		builder := base.NewBalancerBuilder(name, pickerBuilder, base.Config{HealthCheck: true})
		balancer.Register(builder)
		client.balancerBuilder = builder
	}
}
