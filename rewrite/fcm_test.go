package rewrite

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFCMConv(t *testing.T) {
	fcm := FCM{Key: "testkey"}

	request := httptest.NewRequest("POST", "/FCM?token=a", bytes.NewBufferString("content"))
	newReq, err := fcm.Req([]byte("content"), *request)

	require.Nil(t, err, "No error in this test")
	assert.Equal(t, "key=testkey", newReq.Header.Get("Authorization"), "header not set")

	b, _ := ioutil.ReadAll(newReq.Body)
	//							newline bc go
	assert.Equal(t, `{"to":"a","data":{"body":"content"}}`+"\n", string(b), "content not correct")

}
