package rewrite

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type FCM struct {
	Key string
}

func (FCM) Path() string {
	return "/FCM"
}

func (f FCM) Req(body []byte, req http.Request) (newReq *http.Request, err error) {
	token := req.URL.Query().Get("token")

	if len(body) > 1024*4-4 {
		return nil, errors.New("length")
	}

	newBody, err := utils.EncodeJSON(struct {
		To   string            `json:"to"`
		Data map[string]string `json:"data"`
	}{
		To: token,
		Data: map[string]string{
			"body": string(body),
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	newReq, err = http.NewRequest(req.Method, "https://fcm.googleapis.com/fcm/send", newBody)

	//for n, h := range req.Header {
	//	newReq.Header[n] = h
	//}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "key="+f.Key)

	return
}
