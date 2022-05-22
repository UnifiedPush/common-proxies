package rewrite

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
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

func (f FCM) makeReqFromValues(fcmdata fcmData, key string) (newReq *http.Request, err error) {
	newBody, err := utils.EncodeJSON(fcmdata)
	if err != nil {
		fmt.Println(err)
		return nil, err //TODO
	}

	newReq, err = http.NewRequest(http.MethodPost, f.APIURL, newBody)
	if err != nil {
		return nil, err
	}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "key="+key)
	return
}

func (f FCM) Req(body []byte, req http.Request) (requests []*http.Request, error error) {
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
	var data2 map[string]string = nil

	if isV2 {
		b := base64.StdEncoding.EncodeToString(body)
		if len(b) < 3800 {
			data = map[string]string{"b": b, "i": instance}
		} else {
			m := fmt.Sprint(rand.Int63() + 1) // +1 to ensure 0 isn't included
			data = map[string]string{"b": b[:3000], "i": instance, "m": m, "s": "1"}
			data2 = map[string]string{"b": b[3000:], "i": instance, "m": m, "s": "2"}
		}
	} else {
		if app == "" && instance != "" {
			data = map[string]string{"body": string(body), "instance": instance}
		} else if app != "" && instance == "" {
			data = map[string]string{"body": string(body), "app": app}
		} else {
			return nil, utils.NewProxyError(404, errors.New("Invalid query params in v1 FCM"))
		}
	}

	myreq, err := f.makeReqFromValues(fcmData{To: token, Data: data}, key)
	if err != nil {
		return nil, err
	}
	requests = append(requests, myreq)

	if data2 != nil {
		myreq, err := f.makeReqFromValues(fcmData{To: token, Data: data2}, key)
		if err != nil {
			return nil, err
		}
		requests = append(requests, myreq)
	}

	return requests, nil
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
