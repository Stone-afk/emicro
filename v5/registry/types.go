package registry

import (
	"context"
	"io"
)

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
	Instance ServiceInstance
}

//go:generate mockgen -package=mocks -destination=mocks/registry.mock.go -source=types.go Registry
type Registry interface {
	io.Closer
	Register(ctx context.Context, ins ServiceInstance) error
	Unregister(ctx context.Context, ins ServiceInstance) error
	ListServices(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string) (<-chan Event, error)
}
