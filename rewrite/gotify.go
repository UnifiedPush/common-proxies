package rewrite

import (
	"fmt"
	"net/http"

	. "github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/utils"
)

func Gotify(body []byte, req http.Request) (newReq *http.Request, defaultResp *http.Response, err error) {

	url := *req.URL
	url.Scheme = Config.Rewrite.Gotify.Scheme
	url.Host = Config.Rewrite.Gotify.Address
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
