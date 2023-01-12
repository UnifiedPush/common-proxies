package gateway

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type WebPush struct {
	Enabled bool `env:"UP_GATEWAY_WEBPUSH_ENABLE"`
}

func (m WebPush) Path() string {
	if m.Enabled {
		return "/webpush"
	}
	return ""
}

func (m WebPush) Get() []byte {
	return []byte(``)
}

func (m WebPush) Req(body []byte, req http.Request) ([]*http.Request, error) {
	myurl := req.URL.Query().Get("fwdurl")

	wperr := func(err error) *utils.ProxyError {
		return utils.NewProxyError(400, errors.New("not WebPush "+err.Error()))
	}

	switch req.Header.Get("content-encoding") {
	case "aesgcm":
		pubkey := req.Header.Get("crypto-key")
		salt := req.Header.Get("encryption")

		if len(pubkey) < 65 || len(salt) < 16 { // approx values
			return nil, wperr(errors.New("Headers too short"))
		}
		body = append(body, []byte("\n"+pubkey+"\n"+salt+"\naesgcm")...)
		fallthrough
	case "aes128gcm":
		newReq, err := http.NewRequest(http.MethodPost, myurl, bytes.NewReader(body))
		if err != nil {
			return nil, err //TODO
		}
		return []*http.Request{newReq}, nil
	default:
		return nil, wperr(errors.New("Content-Encoding"))
	}
}

func (WebPush) Resp(r []*http.Response, w http.ResponseWriter) {
	w.WriteHeader(r[0].StatusCode)
}

func (m *WebPush) Defaults() (failed bool) {
	return
}
