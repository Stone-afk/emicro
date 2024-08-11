package etcd

import (
	"context"
	"emicro/v5/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"sync"
)

var _ registry.Registry = (*Registry)(nil)

type Registry struct {
	client      *clientv3.Client
	sess        *concurrency.Session
	mutex       sync.RWMutex
	watchCancel []func()
}

func (r *Registry) Close() error {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) Register(ctx context.Context, inst registry.ServiceInstance) error {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) Unregister(ctx context.Context, inst registry.ServiceInstance) error {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) ListServices(ctx context.Context, ServiceName string) ([]registry.ServiceInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) Subscribe(serviceName string) (<-chan registry.Event, error) {
	//TODO implement me
	panic("implement me")
}

func NewRegistry(c *clientv3.Client) (*Registry, error) {
	panic("")
}
