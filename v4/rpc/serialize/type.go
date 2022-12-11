package serialize

// Serializer -> serialization protocol abstract
type Serializer interface {
	Code() byte
	Encode(val any) ([]byte, error)
	Decode(data []byte, val any) error
}
