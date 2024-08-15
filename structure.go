package main

import (
	"net/http"

	"codeberg.org/UnifiedPush/common-proxies/utils"
)

type HttpHandler func(http.ResponseWriter, *http.Request)
type Gateway interface {
	Handler
	Get() []byte
	//Resp make sure to close body in here
	Resp([]*http.Response, http.ResponseWriter)
	Req([]byte, http.Request) ([]*http.Request, error)
}

type Proxy interface {
	Handler
	RespCode(*http.Response) *utils.ProxyError
	Req([]byte, http.Request) ([]*http.Request, error)
}

type Handler interface {
	Path() string
	Defaults() (failed bool)
}
