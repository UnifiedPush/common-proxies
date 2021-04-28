package rewrite

import (
	"fmt"
	"net/http"

	. "github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/utils"
)

func Gotify(body []byte, req http.Request) (newReq *http.Request, defaultResp *http.Response, err error) {

	req.URL.Scheme = Config.Rewrite.Gotify.Scheme
	req.URL.Host = Config.Rewrite.Gotify.Address
	req.URL.Path = "/message"

	newBody, err := utils.EncodeJSON(struct {
		Message string `json:"message"`
	}{
		Message: string(body),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	newReq, err = http.NewRequest(req.Method, req.URL.String(), newBody)

	if err != nil {
		fmt.Println(err)
		return
	}
	//newReq.Header = req.Header
	newReq.Header.Set("Content-Type", "application/json")

	return
}
