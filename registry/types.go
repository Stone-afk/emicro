package registry

import (
	"context"
	"io"
)

type Registry interface {
	Register(ctx context.Context, inst ServiceInstance) error
	UnRegister(ctx context.Context, inst ServiceInstance) error
	ListServices(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string) <-chan Event
	io.Closer
}

type ServiceInstance struct {
	Name    string
	Address string
	Weight  uint32
	Group   string
}

type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypeAdd
	EventTypeDelete
)

type Event struct {
	Type     EventType
	Insrance ServiceInstance
}

type RegistryV1 interface {
	Register(ctx context.Context, ins ServiceInstance) error
	UnRegister(ctx context.Context, ins ServiceInstance) error
	ListServices(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string, listener Listener)
	// Subscribe(listener Listener)
	io.Closer
}

// Listener 给RegistryV1 使用的
type Listener func(event Event) error
