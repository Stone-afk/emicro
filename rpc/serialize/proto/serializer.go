package proto

import (
	"emicro/internal/errs"
	"google.golang.org/protobuf/proto"
)

// Serializer -> Protobuf serialization protocol
type Serializer struct{}

func (s Serializer) Code() byte {
	return 2
}

func (s Serializer) Encode(val any) ([]byte, error) {
	msg, ok := val.(proto.Message)
	if !ok {
		return nil, errs.ProtoSerializeTypError
	}
	return proto.Marshal(msg)
}

func (s Serializer) Decode(data []byte, val any) error {
	msg, ok := val.(proto.Message)
	if !ok {
		return errs.ProtoDeserializeTypError
	}
	return proto.Unmarshal(data, msg)
}
