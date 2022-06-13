package utils

import (
	"errors"
	"fmt"
)

func NewProxyError(code int, err error) *ProxyError {
	return &ProxyError{err, code}
}

func NewProxyErrS(code int, str string, args ...interface{}) *ProxyError {
	return &ProxyError{errors.New(fmt.Sprintf(str, args...)), code}
}

type ProxyError struct {
	S    error
	Code int
}

func (p ProxyError) Error() string {
	return fmt.Sprintf("Error proxying connection: %d because: %s", p.Code, p.S.Error())
}
