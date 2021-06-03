package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
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

	reqs := []*http.Request{}

	for _, i := range pkStruct.Notification.Devices {
		newReq, err := http.NewRequest(http.MethodPost, i.PushKey, bytes.NewReader(body))
		if err != nil {
			return nil, utils.NewProxyError(502, err) //TODO
		}
		reqs = append(reqs, newReq)
	}

	return reqs, nil
}

func (Matrix) Resp(r []*http.Response, w http.ResponseWriter) {
	rejects := struct {
		Rej []string `json:"rejected"`
	}{}
	for _, i := range r {
		if i.StatusCode == 404 {
			rejects.Rej = append(rejects.Rej, i.Request.URL.String())
		}
	}

	b, err := json.Marshal(rejects)
	if err != nil {
		w.WriteHeader(502) //TODO
	}
	w.Write(b)
}
