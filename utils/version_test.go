package utils

import (
	"testing"

	"net/http/httptest"

	"github.com/stretchr/testify/assert"
)

func TestVersionHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	VersionHandler()(rec)

	assert.Equal(t, `{"unifiedpush":{"version":1}}`, rec.Body.String(), "version handler is wrong")
}
