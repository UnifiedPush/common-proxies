package rewrite

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"codeberg.org/UnifiedPush/common-proxies/utils"
	"codeberg.org/UnifiedPush/common-proxies/vapid"
	"golang.org/x/oauth2"
)

type WebPushFCMConfigFactory func(credentialsPath string) (config *FCMConfig, error error)

type WebPushFCMConfig struct {
	TokenSource oauth2.TokenSource
	ApiUrl      string
}

type WebPushFCM struct {
	Enabled         bool   `env:"UP_REWRITE_WEBPUSH_FCM_ENABLE"`
	CredentialsPath string `env:"UP_REWRITE_WEBPUSH_FCM_CREDENTIALS_PATH"`
	privateKey      ecdsa.PrivateKey
	auth            string ""
}

func (f *WebPushFCM) Load() (err error) {
	if !f.Enabled {
		return
	}
	b, err := os.ReadFile(f.CredentialsPath)
	if err != nil {
		log.Println("Cannot read " + f.CredentialsPath)
		return
	}
	private, err := vapid.DecodePriv(b)
	if err != nil {
		log.Println("Cannot decode privkey")
		return
	}
	f.privateKey = *private
	pubkey, err := vapid.EncodePub(private.PublicKey)
	if err != nil {
		log.Println("Cannot encode pubkey")
		return
	}
	log.Println("WebPushFCM PublicKey: " + pubkey)
	auth, err := vapid.GenAuth(rand.Reader, f.privateKey, "https://fcm.googleapis.com", int(time.Now().Add(2*time.Hour).Unix()))
	if err != nil {
		log.Println("Cannot gen new auth !! ", err)
		return
	}
	f.auth = auth
	return
}

func (f WebPushFCM) Path() string {
	if f.Enabled {
		return "/wpfcm"
	}
	return ""
}

func (f WebPushFCM) Duration() time.Duration {
	return 30 * time.Minute
}

func (f *WebPushFCM) Tick() {
	auth, err := vapid.GenAuth(rand.Reader, f.privateKey, "https://fcm.googleapis.com", int(time.Now().Add(2*time.Hour).Unix()))
	if err != nil {
		log.Println("Cannot gen new auth !! ", err)
		return
	}
	fmt.Println("New auth " + auth)
	f.auth = auth
}

// Adds TTL and Content-Encoding headers if not present, and VAPID authorization
func (f WebPushFCM) Req(body []byte, req http.Request) (requests []*http.Request, error error) {
	token := req.URL.Query().Get("t")
	res, _ := regexp.MatchString("^[a-zA-Z0-9-_=:]*$", token)
	if !res {
		return nil, utils.NewProxyError(500, fmt.Errorf("Token not valid"))
	}
	url := fmt.Sprintf("https://fcm.googleapis.com/fcm/send/%s", token)
	//url := fmt.Sprintf("http://127.0.0.1:8000/fcm/send/%s", token)
	newReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if val := req.Header.Get("TTL"); val != "" {
		newReq.Header.Set("TTL", val)
	} else {
		newReq.Header.Set("TTL", "86400") // Cache for a day max
	}
	if val := req.Header.Get("Urgency"); val != "" {
		newReq.Header.Set("Urgency", val)
	} else {
		newReq.Header.Set("Urgency", "normal")
	}
	if val := req.Header.Get("Content-Encoding"); val != "" {
		newReq.Header.Set("Content-Encoding", val)
	} else {
		newReq.Header.Set("Content-Encoding", "aes128gcm")
	}
	newReq.Header.Set("Authorization", f.auth)
	if err != nil {
		return nil, err
	}
	requests = []*http.Request{newReq}
	return
}

func (f WebPushFCM) RespCode(resp *http.Response) *utils.ProxyError {
	return utils.NewProxyErrS(resp.StatusCode, "")
}

func (f *WebPushFCM) Defaults() (failed bool) {
	if !f.Enabled {
		return
	}
	if len(f.CredentialsPath) == 0 {
		log.Println("WebPushFCM Credentials path cannot be empty")
		failed = true
	}
	return
}
