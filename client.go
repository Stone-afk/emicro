package emicro

import (
	"context"
	"emicro/registry"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"time"
)

type ClientOption func(client *Client)

//func ClientWithRegistry(r registry.Registry, timeout time.Duration) ClientOption {
//	return func(client *Client) {
//		client.rb = NewResolverBuilder(r, timeout)
//	}
//}

func ClientWithPickerBuilder(name string, pickerBuilder base.PickerBuilder) ClientOption {
	return func(client *Client) {
		builder := base.NewBalancerBuilder(name, pickerBuilder, base.Config{HealthCheck: true})
		balancer.Register(builder)
		client.balancerBuilder = builder
	}
}

func ClientWithRegistry(r registry.Registry, timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.registry = r
		c.registryTimeout = timeout
	}
}

func ClientWithInsecure() ClientOption {
	return func(c *Client) {
		c.insecure = true
	}
}

type Client struct {
	insecure bool
	//rb       resolver.Builder
	registry        registry.Registry
	registryTimeout time.Duration
	balancerBuilder balancer.Builder
}

func NewClient(opts ...ClientOption) *Client {
	res := &Client{}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (c *Client) Dial(ctx context.Context, service string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	//opts := []grpc.DialOption{grpc.WithResolvers(c.rb)}
	if c.registry != nil {
		rb := NewResolverBuilder(c.registry, c.registryTimeout)
		opts = append(opts, grpc.WithResolvers(rb))
	}
	if c.insecure {
		opts = append(opts, grpc.WithInsecure())
	}
	if len(dialOptions) > 0 {
		opts = append(opts, dialOptions...)
	}
	return grpc.DialContext(ctx, fmt.Sprintf("registry:///%s", service), opts...)
}
