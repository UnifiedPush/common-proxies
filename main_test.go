package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
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
	s.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Call = r
		s.CallBody, _ = ioutil.ReadAll(r.Body)
		w.WriteHeader(200)
		w.Write(s.resp)
	}))

	u, _ := url.Parse(s.ts.URL)
	config.Config.Gateway.AllowedHosts = []string{u.Host}
	ForceBypassValidProviderCheck(s.ts.URL, "http://temp.test")

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

const goodFCMResponse = `{"Results": [{"Error":""}]}`

func (s *RewriteTests) TestFCM() {
	fcm := rewrite.FCM{Key: "testkey", APIURL: s.ts.URL}
	rand.Seed(0)

	myFancyContent, myFancyContent64 := myFancyContentGenerate()
	cases := [][]string{
		{"EFCMD", "/?token=a&instance=b", `{"to":"a","data":{"body":"content","instance":"b"}}`, `content`},
		{"FCMD", "/?token=a&app=a", `{"to":"a","data":{"app":"a","body":"content"}}`, `content`},
		{"FCMv2", "/?token=a&instance=myinst&v2", `{"to":"a","data":{"b":"Y29udGVudA==","i":"myinst"}}`, `content`},
		{"FCMv2-2", "/?v2&token=a&instance=myinst", `{"to":"a","data":{"b":"Y29udGVudA==","i":"myinst"}}`, `content`},
		{"FCMv2-3", "/?v2&token=a&instance=myinst", `{"to":"a","data":{"b":"` + myFancyContent64[3000:] + `","i":"myinst","m":"8717895732742165506","s":"2"}}`, myFancyContent}, // this test only tests the second value because that's much easier than testing for the first one due to the architecture of this file. Someday I'll fix that TODO.
	}

	for _, i := range cases {
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[1], bytes.NewBufferString(i[3])))

		s.Require().Equal(201, s.Resp.Result().StatusCode, "request should be valid")

		s.JSONEq(i[2], string(s.CallBody), "Wrong Content")

		s.Require().NotNil(s.Call, "No request made")
		//call
		s.Equal("key=testkey", s.Call.Header.Get("Authorization"), "header not set")
		if s.T().Failed() {
			println("that was " + i[0])
		}

		s.resetTest()
	}
}

func (s *RewriteTests) TestFCMKeys() {
	fcm := rewrite.FCM{Key: "testkey", APIURL: s.ts.URL, Keys: map[string]string{"1.invalid": "key2", "2.invalid": "key3"}}

	cases := [][]string{
		{"http://example.invalid?v2", "testkey"},
		{"http://1.invalid?v2", "key2"},
		{"http://2.invalid?v2", "key3"},
		{"http://random.test?v2", "testkey"},
	}
	for _, i := range cases {
		s.resetTest()
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[0], bytes.NewBufferString("content")))
		s.Equal("key="+i[1], s.Call.Header.Get("Authorization"), "header not set")
		s.Equal(201, s.Resp.Result().StatusCode, "request should be valid")
	}
	s.Call = nil
	s.Resp = httptest.NewRecorder()

	// This case is where there is no 'default' key, only host specific keys
	//Key omitted for testing
	fcm = rewrite.FCM{APIURL: s.ts.URL, Keys: map[string]string{"1.invalid": "key2", "2.invalid": "key3"}}

	cases = [][]string{
		{"http://1.invalid?v2", "key2"},
		{"http://2.invalid?v2", "key3"},
	}
	for _, i := range cases {
		s.resetTest()
		s.resp = []byte(goodFCMResponse)
		handle(&fcm)(s.Resp, httptest.NewRequest("POST", i[0], bytes.NewBufferString("content")))
		s.Equal("key="+i[1], s.Call.Header.Get("Authorization"), "header not set")
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

func (s *RewriteTests) TestGotify() {
	testurl, _ := url.Parse(s.ts.URL)
	gotify := rewrite.Gotify{Address: testurl.Host, Scheme: testurl.Scheme}

	request := httptest.NewRequest("POST", "/?token=a", bytes.NewBufferString("content"))
	handle(&gotify)(s.Resp, request)

	//resp
	s.Equal(201, s.Resp.Result().StatusCode, "request should be valid")

	s.Require().NotNil(s.Call, "No request made")
	//call
	s.Equal("application/json", s.Call.Header.Get("Content-Type"), "header not set")

	s.Equal(`{"message":"content"}`+"\n", string(s.CallBody), "request body incorrect")
}

func (s *RewriteTests) TestMatrixSend() {
	matrix := gateway.Matrix{}

	content := `{"notification":{"devices":[{"pushkey":"` + s.ts.URL + `"},{"pushkey":"http://temp.test"}], "counts":{"unread":1}}}`
	request := httptest.NewRequest("POST", "/", bytes.NewBufferString(content))
	handle(&matrix)(s.Resp, request)

	//resp
	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	body, _ := ioutil.ReadAll(s.Resp.Body)
	s.Equal(string(body), `{"rejected":[]}`)

	s.Require().NotNil(s.Call, "No request made")

	//call
	s.Equal(`{"notification":{"counts":{"unread":1}}}`, string(s.CallBody), "request body incorrect")

}

func (s *RewriteTests) TestMatrixResp() {
	//TODO
}

func (s *RewriteTests) TestAesgcmGateway() {
	gw := gateway.Aesgcm{}

	content := `this is   
	
my msg`
	request := httptest.NewRequest("POST", "/aesgcm/"+base64.RawURLEncoding.EncodeToString([]byte(s.ts.URL)), bytes.NewBufferString(content))
	request.Header.Add("cOntent-Encoding", "aesgcm")
	request.Header.Add("cryPTo-KEY", `dh="BNoRDbb84JGm8g5Z5CFxurSqsXWJ11ItfXEWYVLE85Y7CYkDjXsIEc4aqxYaQ1G8BqkXCJ6DPpDrWtdWj_mugHU"`)
	request.Header.Add("EncRYPTION", `salt="lngarbyKfMoi9Z75xYXmkg"`)
	handle(&gw)(s.Resp, request)

	s.Equal(200, s.Resp.Result().StatusCode, "request should be valid")
	s.Equal(`aesgcm
Encryption: salt="lngarbyKfMoi9Z75xYXmkg"
Crypto-Key: dh="BNoRDbb84JGm8g5Z5CFxurSqsXWJ11ItfXEWYVLE85Y7CYkDjXsIEc4aqxYaQ1G8BqkXCJ6DPpDrWtdWj_mugHU"
this is   
	
my msg`, string(s.CallBody), "body should match")

}

func (s *RewriteTests) TestHealth() {
	resp, err := http.Get(s.ts.URL + "/health")
	s.Require().Nil(err)
	s.Equal(200, resp.StatusCode)
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
