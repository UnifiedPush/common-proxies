package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/gateway"
	"github.com/karmanyaahm/up_rewrite/rewrite"
	"github.com/stretchr/testify/suite"
)

func init() {
	config.Config.Verbose = true
}

func TestRewriteProxies(t *testing.T) {
	suite.Run(t, new(RewriteTests))
}

type RewriteTests struct {
	suite.Suite
	Call     *http.Request
	CallBody []byte
	Resp     *httptest.ResponseRecorder
	ts       *httptest.Server
}

func (s *RewriteTests) SetupTest() {
	s.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Call = r
		s.CallBody, _ = ioutil.ReadAll(r.Body)
		w.WriteHeader(200)
	}))

	u, _ := url.Parse(s.ts.URL)
	config.Config.Gateway.AllowedHosts = []string{u.Host}
	s.Resp = httptest.NewRecorder()
}

func (s *RewriteTests) TearDownTest() {
	s.ts.Close()
}

func (s *RewriteTests) TestFCM() {
	fcm := rewrite.FCM{Key: "testkey", APIURL: s.ts.URL}

	request := httptest.NewRequest("POST", "/?token=a", bytes.NewBufferString("content"))
	handle(&fcm)(s.Resp, request)

	//resp
	s.Equal(202, s.Resp.Result().StatusCode, "request should be valid")

	s.Require().NotNil(s.Call, "No request made")
	//call
	s.Equal("key=testkey", s.Call.Header.Get("Authorization"), "header not set")

}

func (s *RewriteTests) TestGotify() {
	testurl, _ := url.Parse(s.ts.URL)
	gotify := rewrite.Gotify{Address: testurl.Host, Scheme: testurl.Scheme}

	request := httptest.NewRequest("POST", "/?token=a", bytes.NewBufferString("content"))
	handle(&gotify)(s.Resp, request)

	//resp
	s.Equal(202, s.Resp.Result().StatusCode, "request should be valid")

	s.Require().NotNil(s.Call, "No request made")
	//call
	s.Equal("application/json", s.Call.Header.Get("Content-Type"), "header not set")

	s.Equal(`{"message":"content"}`+"\n", string(s.CallBody), "request body incorrect")
}

func (s *RewriteTests) TestMatrixSend() {
	matrix := gateway.Matrix{}

	content := `{"notification":{"devices":[{"pushkey":"` + s.ts.URL + `"}]}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := ioutil.ReadAll(s.Resp.Body)
	s.Equal(string(body), `{"rejected":[]}`)

	s.Require().NotNil(s.Call, "No request made")

	//call
	s.Equal(content, string(s.CallBody), "request body incorrect")

}

func (s *RewriteTests) TestMatrixResp() {
	//TODO
}
