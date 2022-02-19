package rewrite

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type FCM struct {
	Enabled bool   `env:"UP_REWRITE_FCM_ENABLE"`
	Key     string `env:"UP_REWRITE_FCM_KEY"`
	Keys    map[string]string
	APIURL  string
}

func (f FCM) Path() string {
	if f.Enabled {
		return "/FCM"
	}
	return ""
}

type fcmData struct {
	To   string            `json:"to"`
	Data map[string]string `json:"data"`
}

func (f FCM) Req(body []byte, req http.Request) (*http.Request, error) {
	token := req.URL.Query().Get("token")
	instance := req.URL.Query().Get("instance")
	app := req.URL.Query().Get("app")
	isV2 := req.URL.Query().Has("v2")

	key := f.Key
	if k, ok := f.Keys[req.Host]; ok {
		key = k
	} else if key == "" {
		return nil, utils.NewProxyError(404, errors.New("Endpoint doesn't exist. Wrong Host "+req.Host))
	}

	var data map[string]string

	if isV2 {
		//it's a little under 3072 but 3072 is def over. i'll test the specifics later
		if len(body) > 3072 {
			return nil, utils.NewProxyError(413, errors.New("FCM Payload length medium"))
		}
		data = map[string]string{"b": base64.StdEncoding.EncodeToString(body), "i": instance}
	} else {
		if app == "" && instance != "" {
			data = map[string]string{"body": string(body), "instance": instance}
		} else if app != "" && instance == "" {
			data = map[string]string{"body": string(body), "app": app}
		} else {
			return nil, utils.NewProxyError(404, errors.New("Invalid query params in v1 FCM"))
		}
	}

	newBody, err := utils.EncodeJSON(fcmData{
		To:   token,
		Data: data,
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
	newReq.Header.Set("Authorization", "key="+key)

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
	if len(f.Key) == 0 && len(f.Keys) == 0 {
		log.Println("FCM Key cannot be empty")
		failed = true
	}
	f.APIURL = "https://fcm.googleapis.com/fcm/send"
	return
}
