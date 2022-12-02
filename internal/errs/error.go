package errs

import (
	"errors"
	"fmt"
)

var (
	ServiceTypError     = errors.New("emicro: service type must be a first level pointer")
	ReadLenDataError    = errors.New("emicro: could not read the length data")
	ReadRespFailError   = errors.New("emicro: unable to read response")
	InvalidServiceName  = errors.New("emicro: Invalid service name")
	ClientNotAllWritten = errors.New("emicro: client not all data is written")
	OnewayError         = errors.New("emicro: 这是 oneway 调用")
)

var (
	ProtoSerializeTypError   = errors.New("serialize: serialization must be proto Message Type")
	ProtoDeserializeTypError = errors.New("serialize: deserialization must be proto.Message type")
)

func ServerResponseFailed(err error) error {
	return fmt.Errorf("emicro: server sending response failed: %v", err)
}

func ClientConnDeaded(err error) error {
	return fmt.Errorf("emicro: client unable to get an available connection %w", err)
}
