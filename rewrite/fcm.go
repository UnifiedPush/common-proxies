package rewrite

import (
	"fmt"
	"log"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type FCM struct {
	Enabled bool   `env:"UP_REWRITE_FCM_ENABLE"`
	Key     string `env:"UP_REWRITE_FCM_KEY"`
	APIURL  string
}

func (f FCM) Path() string {
	if f.Enabled {
		return "/FCM"
	}
	return ""
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

func (f *FCM) Defaults() (failed bool) {
	if !f.Enabled {
		return
	}
	if len(f.Key) == 0 {
		log.Println("FCM Key cannot be empty")
		failed = true
	}
	f.APIURL = "https://fcm.googleapis.com/fcm/send"
	return
}
