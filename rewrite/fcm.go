package rewrite

import (
	"encoding/base64"
	"encoding/json"
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

type fcmResp struct {
	Results []struct {
		Error string
	}
}

func (f FCM) RespCode(resp *http.Response) int {
	switch resp.StatusCode / 100 {
	case 4: // 4xx
		// error in common-proxies, not app server
		return 500
	case 5: // 5xx
		//delay, TODO implement forced exponential backoff in common-proxies
		return 429
	}

	dec := json.NewDecoder(resp.Body)

	out := fcmResp{}
	err := dec.Decode(&out)
	if err != nil || len(out.Results) < 1 {
		// already established it's not a 401, so dunno whats going on
		return 502
	}

	givenErr := out.Results[0].Error

	fmt.Printf("%s %d\n", givenErr, resp.StatusCode)
	switch givenErr {
	case "MissingRegistration", "InvalidRegistration", "NotRegistered", "MismatchSenderId":
		return 404
		//case "InvalidParameters": // doesn't happen because 4xx is handled above
	case "MessageTooBig", "InvalidDataKey", "InvalidTtl", "TopicsMessageRateExceeded", "InvalidApnsCredential": // this shouldn't happen, common-proxies has its own checks for size, common-proxies controls the keys, common-proxies doesn't send TTL, common-proxies doesn't deal in topics, idk apns
		return 502
	case "Unavailable", "InternalServerError", "DeviceMessageRateExceeded":
		//delay, TODO implement forced exponential backoff
		return 429
	default:
		return 201
	}
	//TODO log
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
