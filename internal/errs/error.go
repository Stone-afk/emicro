package errs

import "errors"

var (
	ServiceTypError = errors.New("服务类型必须为一级指针")
)
