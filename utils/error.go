package utils

import "fmt"

func NewProxyError(code int, err error) *ProxyError {
	return &ProxyError{err, code}
}

type ProxyError struct {
	s    error
	Code int
}

func (p *ProxyError) Error() string {
	return fmt.Sprint("Error proxying connection: %d because: ", p.Code, p.s.Error())
}
