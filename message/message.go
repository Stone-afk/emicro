package message

// Request ->
type Request struct {
	ServiceName string
	Method      string
	Data        []byte
}

// Response ->
type Response struct {
	Error string
	Data  []byte
}
