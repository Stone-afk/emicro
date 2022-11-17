//go:build v1

package emicro

type Request struct {
	// for scalability
	ServiceName string
	Method      string
	// request itself
	Data []byte
}

type Response struct {
	Data  []byte
	Error string
}
