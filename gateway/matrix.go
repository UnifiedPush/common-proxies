package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type Matrix struct {
}

func (m Matrix) Path() string {
	return "/_matrix/push/v1/notify"
}

func (m Matrix) Get() []byte {
	return []byte(`{"gateway":"matrix","unifiedpush":{"gateway":"matrix"}}`)
}

func (m Matrix) Req(body []byte, req http.Request) ([]*http.Request, *utils.ProxyError) {
	pkStruct := struct {
		Notification struct {
			Devices []struct {
				PushKey string
			}
		}
	}{}
	json.Unmarshal(body, &pkStruct)
	if !(len(pkStruct.Notification.Devices) > 0) {
		return nil, utils.NewProxyError(400, errors.New("Gateway URL"))
	}
	pushKey := pkStruct.Notification.Devices[0].PushKey

	newReq, err := http.NewRequest(req.Method, pushKey, bytes.NewReader(body))
	if err != nil {
		return nil, utils.NewProxyError(502, err) //TODO
	}

	newReq.Header.Set("Content-Type", "application/json")
	return []*http.Request{newReq}, nil
}

func (Matrix) Resp(r *http.Response) {
	r.Body = ioutil.NopCloser(bytes.NewBufferString(`{}`))
}
