package rewrite

import (
	"fmt"
	"net/http"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type Gotify struct {
	Address string
	Scheme  string
}

func (Gotify) Path() string {
	return "/UP"
}

func (g Gotify) Req(body []byte, req http.Request) (newReq *http.Request, err error) {

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
		fmt.Println(err)
		return
	}

	newReq, err = http.NewRequest(req.Method, url.String(), newBody)

	if err != nil {
		fmt.Println(err)
		return
	}
	//newReq.Header = req.Header
	newReq.Header.Set("Content-Type", "application/json")

	return
}
