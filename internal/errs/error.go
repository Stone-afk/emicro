package errs

import "errors"

var (
	ServiceTypError    = errors.New("emicro: service type must be a first level pointer")
	ReadLenDataError   = errors.New("emicro: could not read the length data")
	ReadRespFailError  = errors.New("emicro: unable to read response")
	InvalidServiceName = errors.New("emicro: Invalid service name")
)

var (
	ProtoSerializeTypError   = errors.New("serialize: serialization must be proto Message Type")
	ProtoDeserializeTypError = errors.New("serialize: deserialization must be proto.Message type")
)
