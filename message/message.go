package message

type Request struct {
	ServiceName string
	Method      string
	Data        []byte
}

type Response struct {
	Error string
	Data  []byte
}
