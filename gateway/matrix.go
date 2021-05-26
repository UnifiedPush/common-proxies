package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Matrix struct {
}

func (m Matrix) Path() string {
	return "/_matrix/push/v1/notify"
}

func (m Matrix) Get() []byte {
	return []byte(`{"gateway":"matrix","unifiedpush":{"gateway":"matrix"}}`)
}

func (m Matrix) Req(body []byte, req http.Request) (newReq *http.Request, err error) {
	pkStruct := struct {
		Notification struct {
			Devices []struct {
				PushKey string
			}
		}
	}{}
	json.Unmarshal(body, &pkStruct)
	if !(len(pkStruct.Notification.Devices) > 0) {
		return nil, errors.New("Gateway URL")
	}
	pushKey := pkStruct.Notification.Devices[0].PushKey

	newReq, err = http.NewRequest(req.Method, pushKey, bytes.NewReader(body))
	if err != nil {
		fmt.Println(err)
		newReq = nil
		return
	}

	newReq.Header.Set("Content-Type", "application/json")
	return
}

func (Matrix) Resp(r *http.Response) {
	r.Body = ioutil.NopCloser(bytes.NewBufferString(`{}`))
}
