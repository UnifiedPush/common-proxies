package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
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
		s.CallBody, _ = ioutil.ReadAll(r.Body)
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

const goodFCMResponse = `{"Results": [{"Error":""}]}`

func (s *RewriteTests) TestFCM() {
	fcm := rewrite.FCM{
		CredentialsPath: "testproject",
		ConfigFactory:   testConfigFactory(s.ts.URL),
	}
	rand.Seed(0)

	myFancyContent, myFancyContent64 := myFancyContentGenerate()
	cases := [][]string{
		{"EFCMD", "/?token=a&instance=b", `{"message":{"token":"a","data":{"body":"content","instance":"b"}}}`, `content`},
		{"FCMD", "/?token=a&app=a", `{"message":{"token":"a","data":{"app":"a","body":"content"}}}`, `content`},
		{"FCMv2", "/?token=a&instance=myinst&v2", `{"message":{"token":"a","data":{"b":"Y29udGVudA==","i":"myinst"}}}`, `content`},
		{"FCMv2-2", "/?v2&token=a&instance=myinst", `{"message":{"token":"a","data":{"b":"Y29udGVudA==","i":"myinst"}}}`, `content`},
		{"FCMv2-3", "/?v2&token=a&instance=myinst", `{"message":{"token":"a","data":{"b":"` + myFancyContent64[3000:] + `","i":"myinst","m":"8717895732742165506","s":"2"}}}`, myFancyContent}, // this test only tests the second value because that's much easier than testing for the first one due to the architecture of this file. Someday I'll fix that TODO.
	}

	for _, i := range cases {
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[1], bytes.NewBufferString(i[3])))

		s.Require().Equal(201, s.Resp.Result().StatusCode, "request should be valid")

		s.JSONEq(i[2], string(s.CallBody), "Wrong Content")

		s.Require().NotNil(s.Call, "No request made")
		//call
		s.Equal("Bearer faketoken_testproject", s.Call.Header.Get("Authorization"), "header not set")
		if s.T().Failed() {
			println("that was " + i[0])
		}

		s.resetTest()
	}
}

func (s *RewriteTests) TestFCMCredentialsPaths() {
	fcm := rewrite.FCM{
		CredentialsPath:  "testproject",
		ConfigFactory:    testConfigFactory(s.ts.URL),
		CredentialsPaths: map[string]string{"1.invalid": "project2", "2.invalid": "project3"},
	}

	cases := [][]string{
		{"http://example.invalid?v2", "testproject"},
		{"http://1.invalid?v2", "project2"},
		{"http://2.invalid?v2", "project3"},
		{"http://random.test?v2", "testproject"},
	}
	for _, i := range cases {
		s.resetTest()
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[0], bytes.NewBufferString("content")))
		s.Equal("Bearer faketoken_"+i[1], s.Call.Header.Get("Authorization"), "header not set")
		s.Equal(201, s.Resp.Result().StatusCode, "request should be valid")
	}
	s.Call = nil
	s.Resp = httptest.NewRecorder()

	// This case is where there is no 'default' key, only host specific keys
	//ProjectID omitted for testings
	fcm = rewrite.FCM{
		ConfigFactory:    testConfigFactory(s.ts.URL),
		CredentialsPaths: map[string]string{"1.invalid": "project2", "2.invalid": "project3"},
	}

	cases = [][]string{
		{"http://1.invalid?v2", "project2"},
		{"http://2.invalid?v2", "project3"},
	}
	for _, i := range cases {
		s.resetTest()
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[0], bytes.NewBufferString("content")))
		s.Equal("Bearer faketoken_"+i[1], s.Call.Header.Get("Authorization"), "header not set")
		s.Equal(201, s.Resp.Result().StatusCode, "request should be valid")
	}
	s.Call = nil
	s.Resp = httptest.NewRecorder()

	cases = [][]string{
		{"http://123.invalid?v2"},
		{"http://random.invalid?v2"},
	}
	for _, i := range cases {
		s.resetTest()
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[0], bytes.NewBufferString("content")))

		s.Nil(s.Call, "no request should be made because of the error")
		s.Equal(404, s.Resp.Result().StatusCode, "request should be invalid")
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
	s.Equal(`{"rejected":["`+url+`"]}`, string(body))
	u, _ := neturl.Parse(url)
	s.Equal(Refused, getEndpointStatus(u))
}

func (s *RewriteTests) TestMatrixRejectedLookupNoHost() {
	matrix := gateway.Matrix{}

	url := "http://aaaa"
	content := `{"notification":{"devices":[{"pushkey":"` + url + `"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := ioutil.ReadAll(s.Resp.Body)
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
	body, _ := ioutil.ReadAll(s.Resp.Body)
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

func myFancyContentGenerate() (string, string) {
	myFancyContent := []byte{}
	for i := 0; i < 4096; i++ {
		myFancyContent = append(myFancyContent, byte(i%256))
	}

	return string(myFancyContent), "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"
}
