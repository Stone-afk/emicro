package errs

import "errors"

var (
	ServiceTypError    = errors.New("service type must be a first level pointer")
	ReadLenDataError   = errors.New("could not read the length data")
	ReadRespFailError  = errors.New("unable to read response")
	InvalidServiceName = errors.New("Invalid service name")
)
