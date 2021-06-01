package rewrite

import (
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

func (f FCM) Req(body []byte, req http.Request) (*http.Request, *utils.ProxyError) {
	token := req.URL.Query().Get("token")

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
		return nil, utils.NewProxyError(502, err) //TODO
	}

	newReq, err := http.NewRequest(http.MethodPost, "https://fcm.googleapis.com/fcm/send", newBody)
	if err != nil {
		return nil, utils.NewProxyError(502, err)
	}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "key="+f.Key)

	return newReq, nil
}
