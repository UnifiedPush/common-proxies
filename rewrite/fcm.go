package rewrite

/**
/!\ Important: this way to send FCM messages are deprecated, please use wp_fcm instead.
*/

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"codeberg.org/UnifiedPush/common-proxies/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type FCMConfigFactory func(credentialsPath string) (config *FCMConfig, error error)

type FCMConfig struct {
	TokenSource oauth2.TokenSource
	ApiUrl      string
}

type FCM struct {
	Enabled          bool   `env:"UP_REWRITE_FCM_ENABLE"`
	CredentialsPath  string `env:"UP_REWRITE_FCM_CREDENTIALS_PATH"`
	CredentialsPaths map[string]string
	ConfigFactory    FCMConfigFactory
}

var googleConfigs = map[string]FCMConfig{}
var googleConfigsLock = sync.RWMutex{}

func googleConfigFactory(credentialsPath string) (config *FCMConfig, error error) {
	googleConfigsLock.Lock()
	defer googleConfigsLock.Unlock()

	existing, exists := googleConfigs[credentialsPath]
	if exists {
		return &existing, nil
	}

	jsonData, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, utils.NewProxyError(500, errors.New("could not load credentials file"))
	}

	conf, err := google.CredentialsFromJSON(context.Background(), jsonData, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		log.Println(err)
		return nil, utils.NewProxyError(500, errors.New("could not create FCM credential source"))
	}

	source := FCMConfig{
		TokenSource: oauth2.ReuseTokenSource(nil, conf.TokenSource),
		ApiUrl:      fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", conf.ProjectID),
	}
	googleConfigs[credentialsPath] = source
	return &source, nil
}

func (f FCM) Load() (err error) {
	// TODO: load config once
	log.Println(`

			!! This way to send FCM messages is deprecated !!
			Please use wp_fcm instead.

		`)
	return
}

func (f FCM) Path() string {
	if f.Enabled {
		return "/FCM"
	}
	return ""
}

type fcmData struct {
	Token string            `json:"token"`
	Data  map[string]string `json:"data"`
}

type fcmPayload struct {
	Message fcmData `json:"message"`
}

func (f FCM) makeReqFromValues(data fcmData, config *FCMConfig) (newReq *http.Request, err error) {
	newBody, err := utils.EncodeJSON(fcmPayload{Message: data})
	if err != nil {
		fmt.Println(err)
		return nil, err //TODO
	}

	newReq, err = http.NewRequest(http.MethodPost, config.ApiUrl, newBody)
	if err != nil {
		return nil, err
	}

	token, err := config.TokenSource.Token()

	if err != nil {
		return nil, err
	}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return
}

func (f FCM) Req(body []byte, req http.Request) (requests []*http.Request, error error) {
	token := req.URL.Query().Get("token")
	instance := req.URL.Query().Get("instance")
	app := req.URL.Query().Get("app")
	isV2 := req.URL.Query().Has("v2")

	credentialsPath := f.CredentialsPath
	if path, ok := f.CredentialsPaths[req.Host]; ok {
		credentialsPath = path
	} else if credentialsPath == "" {
		return nil, utils.NewProxyError(404, errors.New("Endpoint doesn't exist. Wrong Host "+req.Host))
	}

	config, err := f.ConfigFactory(credentialsPath)

	if err != nil {
		return nil, utils.NewProxyError(500, err)
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

	myreq, err := f.makeReqFromValues(fcmData{Token: token, Data: data}, config)
	if err != nil {
		return nil, err
	}
	requests = append(requests, myreq)

	if data2 != nil {
		myreq, err := f.makeReqFromValues(fcmData{Token: token, Data: data2}, config)
		if err != nil {
			return nil, err
		}
		requests = append(requests, myreq)
	}

	return requests, nil
}

type fcmResp struct {
	Name string
}

type fcmErr struct {
	Message string
}

func (f FCM) RespCode(resp *http.Response) *utils.ProxyError {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 5000))
	switch resp.StatusCode / 100 {
	case 4: // 4xx
		out := fcmErr{}
		err := json.Unmarshal(b, &out)
		if err != nil {
			// Not even to extract err from body
			return utils.NewProxyErrS(500, "Error with common-proxies auth or json, not app server, this should not be happening")
		}
		return utils.NewProxyErrS(resp.StatusCode, "FCM error: %s", out.Message)
	case 5: // 5xx
		//TODO implement forced exponential backoff in common-proxies
		return utils.NewProxyErrS(429, "slow down")
	}

	out := fcmResp{}
	err := json.Unmarshal(b, &out)
	if err != nil {
		//
		return utils.NewProxyErrS(502, "dunno whats going on, resp is not json or not in right schema %s", string(b))
	}

	return utils.NewProxyErrS(201, "")
	//TODO log
}

func (f *FCM) Defaults() (failed bool) {
	if !f.Enabled {
		return
	}
	if len(f.CredentialsPath) == 0 && len(f.CredentialsPaths) == 0 {
		log.Println("FCM credentials path cannot be empty")
		failed = true
	}
	f.ConfigFactory = googleConfigFactory
	return
}
