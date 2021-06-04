package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karmanyaahm/up_rewrite/rewrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationFCM(t *testing.T) {
	fcm := rewrite.FCM{Key: "testkey"}
	handler := handle(fcm)

	request := httptest.NewRequest("POST", "/FCM?token=a", bytes.NewBufferString("content"))

	resp := httptest.NewRecorder()
	var call *http.Request
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call = r
		ts.Close()
	}))
	fcm.APIURL = ts.URL

	handler(resp, request)

	//resp
	assert.Equal(t, 202, resp.Result().StatusCode, "request should be valid")

	require.NotNil(t, call, "No request made")
	//call
	assert.Equal(t, "key=testkey", call.Header.Get("Authorization"), "header not set")

}

func rewriteTest(rewrite interface{}, req http.Request) (resp httptest.ResponseRecorder, call *http.Request) {
	return
}
