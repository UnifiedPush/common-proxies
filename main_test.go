package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"
	"time"

	"codeberg.org/UnifiedPush/common-proxies/config"
	"codeberg.org/UnifiedPush/common-proxies/gateway"
	"codeberg.org/UnifiedPush/common-proxies/rewrite"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

func init() {
	config.Config.Verbose = true
	config.Defaults(&config.Config)
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
	resp     []byte
}

func (s *RewriteTests) SetupTest() {
	// Setup allowed Web Push endpoint
	s.SetupTestServer(201, true, false)
}

func (s *RewriteTests) SetupTestServer(statusCode int, allowed bool, timeout bool) {
	s.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if timeout {
			time.Sleep(10 * time.Second)
		}
		s.Call = r
		s.CallBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(statusCode)
		w.Write(s.resp)
	}))

	u, _ := neturl.Parse(s.ts.URL)
	if allowed {
		log.Println("TestServer, allowed: ", u)
		config.Config.Gateway.AllowedHosts = []string{u.Host}
	} else {
		log.Println("TestServer, not allowed: ", u)
	}

	s.resetTest()
}

func (s *RewriteTests) resetTest() {
	s.Call = nil
	s.Resp = httptest.NewRecorder()
	s.CallBody = []byte("")
	s.resp = []byte{}
}

func (s *RewriteTests) TearDownTest() {
	s.ts.Close()
}

type FakeTokenSource struct {
	token string
}

func (s FakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken:  "faketoken_" + s.token,
		TokenType:    "fake",
		RefreshToken: "",
	}, nil
}

func testConfigFactory(apiUrl string) rewrite.FCMConfigFactory {
	return func(jsonPath string) (config *rewrite.FCMConfig, error error) {
		return &rewrite.FCMConfig{
			ApiUrl:      apiUrl,
			TokenSource: FakeTokenSource{token: jsonPath},
		}, nil
	}
}

func (s *RewriteTests) TestMatrixAllowed() {
	matrix := gateway.Matrix{}

	url := s.ts.URL
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":[]}`, string(body))

	s.Require().NotNil(s.Call, "No request made")

	//call
	s.Equal(`{"notification":{"counts":{"unread":1}}}`, string(s.CallBody), "request body incorrect")
}

func (s *RewriteTests) TestMatrixRejectedFromCache() {
	u, _ := neturl.Parse(s.ts.URL)
	setEndpointStatus(u, Refused)
	matrix := gateway.Matrix{}

	url := s.ts.URL
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
}

func (s *RewriteTests) TestMatrixRejected404() {
	// Setup allowed web push endpoint unsubscribed
	s.SetupTestServer(404, true, false)
	matrix := gateway.Matrix{}

	url := s.ts.URL
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(Refused, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixRejectedBadIP() {
	// Setup forbiden web push endpoint
	s.SetupTestServer(201, false, false)
	matrix := gateway.Matrix{}

	url := s.ts.URL
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(Refused, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixRejectedUnsupportedProtocol() {
	matrix := gateway.Matrix{}

	url := "unix://foo"
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(Refused, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixRejectedLookupNoHost() {
	matrix := gateway.Matrix{}

	url := "http://doesnotexist.unifiedpush.org"
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	// Nothing in the spec allows to handle temp unavailable
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(Refused, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixRejectedTimeout() {
	// Setup allowed Web Push endpoint who timeout
	s.SetupTestServer(201, true, true)
	matrix := gateway.Matrix{}

	url := s.ts.URL
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := io.ReadAll(s.Resp.Body)
	// Nothing in the spec allows to handle temp unavailable
	s.Equal(`{"rejected":[]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(TemporaryUnavailable, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixResp() {
	//TODO
}

func (s *RewriteTests) TestGenericGateway() {
	gw := gateway.Generic{}

	content := `this is   
	
my msg`
	request := httptest.NewRequest("POST", "/generic/"+base64.RawURLEncoding.EncodeToString([]byte(s.ts.URL)), bytes.NewBufferString(content))
	request.Header.Add("cOntent-Encoding", "aesgcm")
	request.Header.Add("cryPTo-KEY", `dh="BNoRDbb84JGm8g5Z5CFxurSqsXWJ11ItfXEWYVLE85Y7CYkDjXsIEc4aqxYaQ1G8BqkXCJ6DPpDrWtdWj_mugHU"`)
	request.Header.Add("EncRYPTION", `Encryption: salt="lngarbyKfMoi9Z75xYXmkg"`)
	handle(&gw)(s.Resp, request)

	s.Equal(201, s.Resp.Result().StatusCode, "request should be valid")
	s.Equal(`this is   
	
my msg
dh="BNoRDbb84JGm8g5Z5CFxurSqsXWJ11ItfXEWYVLE85Y7CYkDjXsIEc4aqxYaQ1G8BqkXCJ6DPpDrWtdWj_mugHU"
Encryption: salt="lngarbyKfMoi9Z75xYXmkg"
aesgcm`, string(s.CallBody), "body should match")

}

func (s *RewriteTests) TestHealth() {
	resp, err := http.Get(s.ts.URL + "/health")
	s.Require().Nil(err)
	s.Equal(201, resp.StatusCode)
	read, err := io.ReadAll(resp.Body)
	s.Require().Nil(err)
	s.Contains(`OK`, string(read))
}
