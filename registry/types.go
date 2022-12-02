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
