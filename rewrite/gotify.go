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

func (g Gotify) Req(body []byte, req http.Request) (*http.Request, *utils.ProxyError) {

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
		return nil, utils.NewProxyError(502, err)
	}

	newReq, err := http.NewRequest(req.Method, url.String(), newBody)

	if err != nil {
		fmt.Println(err)
		return nil, utils.NewProxyError(502, err) //TODO refine err codes
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
