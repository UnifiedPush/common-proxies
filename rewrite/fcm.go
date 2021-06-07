package rewrite

import (
	"fmt"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type FCM struct {
	Key    string
	APIURL string
}

func (FCM) Path() string {
	return "/FCM"
}

type fcmData struct {
	To       string            `json:"to"`
	Data     map[string]string `json:"data"`
	Instance string            `json:"instance"`
}

func (f FCM) Req(body []byte, req http.Request) (*http.Request, error) {
	token := req.URL.Query().Get("token")
	instance := req.URL.Query().Get("instance")

	newBody, err := utils.EncodeJSON(fcmData{
		To: token,
		Data: map[string]string{
			"body": string(body),
		},
		Instance: instance,
	})
	if err != nil {
		fmt.Println(err)
		return nil, err //TODO
	}

	newReq, err := http.NewRequest(http.MethodPost, f.APIURL, newBody)
	if err != nil {
		return nil, err
	}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "key="+f.Key)

	return newReq, nil
}

func (f FCM) RespCode(resp *http.Response) int {
	return 202
	//TODO https://firebase.google.com/docs/cloud-messaging/http-server-ref?authuser=0#error-codes
}
