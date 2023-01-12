package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type WebPush1 struct {
	Enabled bool `env:"UP_GATEWAY_WP_ENABLE"`
}

func (m WebPush1) Path() string {
	if m.Enabled {
		return "/aesgcmwp1"
	}
	return ""
}

func (m WebPush1) Get() []byte {
	return []byte(``)
}

func (m WebPush1) Req(body []byte, req http.Request) ([]*http.Request, error) {
	myurl := req.URL.Query().Get("fwdurl")
	log.Println(myurl)
	fmt.Println(req.Header)
	fmt.Println(base64.URLEncoding.EncodeToString(body))

	wperr := func(err error) *utils.ProxyError {
		return utils.NewProxyError(400, errors.New("Content Encoding, not WebPush "+err.Error()))
	}

	switch req.Header.Get("content-encoding") {
	case "aesgcm":
		pubkey, err := parseStructure("dh", req.Header.Get("crypto-key"))
		if err != nil {
			log.Println(req.Header.Get("crypto-key"))
			return nil, wperr(err)
		}
		salt, err := parseStructure("salt", req.Header.Get("encryption"))

		if len(pubkey) != 65 || len(salt) != 16 { // approx values
			return nil, wperr(err)
		}

		log.Println(base64.RawStdEncoding.EncodeToString(body))
		// order goes -> salt (16), len(body) (4), len(key) (1), key (65), old body
		salt = binary.BigEndian.AppendUint32(salt, uint32(len(body)))
		salt = append(salt, byte(len(pubkey)))
		salt = append(salt, pubkey...)
		body = append(salt, body...)
		log.Println(base64.RawStdEncoding.EncodeToString(body))
		fallthrough
	case "aes128gcm":
		newReq, err := http.NewRequest(http.MethodPost, myurl, bytes.NewReader(body))
		if err != nil {
			return nil, err //TODO
		}
		return []*http.Request{newReq}, nil
	default:
		return nil, wperr(errors.New("Content-Encoding"))
	}
}

func (WebPush1) Resp(r []*http.Response, w http.ResponseWriter) {
	rejects := struct {
		Rej []string `json:"rejected"`
	}{}
	rejects.Rej = make([]string, 0)
	for _, i := range r {
		if i != nil && i.StatusCode > 400 && i.StatusCode <= 404 {
			rejects.Rej = append(rejects.Rej, i.Request.URL.String())
		}
	}

	b, err := json.Marshal(rejects)
	if err != nil {
		w.WriteHeader(502) //TODO
	}
	w.Write(b)
}

func parseStructure(want, from string) ([]byte, error) {
	_, after, found := strings.Cut(from, want+"=")
	if !found {
		return []byte{}, errors.New("Key not found")
	}
	before, _, _ := strings.Cut(after, ";")
	return base64.RawURLEncoding.Strict().DecodeString(before)
}

func (m *WebPush1) Defaults() (failed bool) {
	return
}
