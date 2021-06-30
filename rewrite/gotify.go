package rewrite

import (
	"log"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type Gotify struct {
	Enabled bool   `env:"UP_REWRITE_GOTIFY_ENABLE"`
	Address string `env:"UP_REWRITE_GOTIFY_ADDRESS"`
	Scheme  string `env:"UP_REWRITE_GOTIFY_SCHEME"`
}

func (g Gotify) Path() string {
	if g.Enabled {
		return "/UP"
	}
	return ""
}

func (g Gotify) Req(body []byte, req http.Request) (*http.Request, error) {

	url := *req.URL
	url.Scheme = g.Scheme
	url.Host = g.Address
	url.Path = "/message"

	newBody, err := utils.EncodeJSON(struct {
		Message string `json:"message"`
	}{
		Message: string(body),
	})
	if err != nil {
		return nil, err
	}

	newReq, err := http.NewRequest(req.Method, url.String(), newBody)

	if err != nil {
		return nil, err //TODO refine err codes
	}
	//newReq.Header = req.Header
	newReq.Header.Set("Content-Type", "application/json")

	return newReq, nil
}

func (g Gotify) RespCode(resp *http.Response) int {
	//convert gotify message response to up resp
	return map[int]int{
		401: 404,
		403: 404,
		400: 502, //unknown how to handle gotify 400 TODO
		200: 202,
	}[resp.StatusCode]
}

func (g *Gotify) Defaults() (failed bool) {
	if !g.Enabled {
		return
	}
	if len(g.Address) == 0 {
		log.Println("Gotify Address cannot be empty")
		failed = true
	}

	g.Scheme = strings.ToLower(g.Scheme)
	if !(g.Scheme == "http" || g.Scheme == "https") {
		log.Println("Gotify Scheme incorrect")
		failed = true
	}
	return
}
