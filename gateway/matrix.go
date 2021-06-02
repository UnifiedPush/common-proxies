package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func Matrix(body []byte, req http.Request) (newReq *http.Request, defaultResp *http.Response, err error) {
	if req.Method == http.MethodGet {
		content := []byte(`{"gateway":"matrix"}`)
		defaultResp = &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader(content)),
		}
		defaultResp.StatusCode = http.StatusOK

		return
	}

	pkStruct := struct {
		Notification struct {
			Devices []struct {
				PushKey string
			}
		}
	}{}
	json.Unmarshal(body, &pkStruct)
	if !(len(pkStruct.Notification.Devices) > 0) {
		return nil, nil, errors.New("Gateway URL")
	}
	pushKey := pkStruct.Notification.Devices[0].PushKey

	newReq, err = http.NewRequest(req.Method, pushKey, bytes.NewReader(body))
	if err != nil {
		fmt.Println(err)
		newReq = nil
		return
	}

	//newReq.Header = req.Header
	newReq.Header.Set("Content-Type", "application/json")
	return
}

func MatrixResp(r *http.Response) {
	r.Body = ioutil.NopCloser(bytes.NewBufferString(`{"rejected":[]}`))
}
