package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type Matrix struct {
	Enabled bool `env:"UP_GATEWAY_MATRIX_ENABLE"`
}

func (m Matrix) Path() string {
	if m.Enabled {
		return "/_matrix/push/v1/notify"
	}
	return ""
}

func (m Matrix) Get() []byte {
	return []byte(`{"gateway":"matrix","unifiedpush":{"gateway":"matrix"}}`)
}

func (m Matrix) Req(body []byte, req http.Request) ([]*http.Request, error) {
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
			return nil, err //TODO
		}
		reqs = append(reqs, newReq)
	}

	return reqs, nil
}

func (Matrix) Resp(r []*http.Response, w http.ResponseWriter) {
	rejects := struct {
		Rej []string `json:"rejected"`
	}{}
	rejects.Rej = make([]string, 0)
	for _, i := range r {
		if i != nil && i.StatusCode > 400 && i.StatusCode <= 404 {
			rejects.Rej = append(rejects.Rej, i.Request.URL.String())
		}
	}

	b, err := json.Marshal(rejects)
	if err != nil {
		w.WriteHeader(502) //TODO
	}
	w.Write(b)
}

func (m *Matrix) Defaults() (failed bool) {
	return
}
