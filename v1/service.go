//go:build v1

package emicro

type Service interface {
	ServiceName() string
}
