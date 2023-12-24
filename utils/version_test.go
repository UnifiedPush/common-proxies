package utils

import (
	"encoding/json"
	"testing"

	"net/http/httptest"

	"github.com/stretchr/testify/assert"
)

func TestVersionHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	VersionHandler()(rec)

	assert.Equal(t, `{"unifiedpush":{"version":1}}`, rec.Body.String(), "version handler is wrong")

	payload, err := json.Marshal(VHandler{UP{1, ""}})
	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, `{"unifiedpush":{"version":1}}`, string(payload), "version handler is wrong")

	payload, err = json.Marshal(VHandler{UP{0, "aesgcm"}})
	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, `{"unifiedpush":{"gateway":"aesgcm"}}`, string(payload), "version handler is wrong")
}
