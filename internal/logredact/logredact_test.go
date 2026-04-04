package logredact_test

import (
	"testing"

	"github.com/freeway-vpn/backend/internal/logredact"
	"github.com/stretchr/testify/assert"
)

func TestHTTPPathForLog(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", logredact.HTTPPathForLog(""))
	assert.Equal(t, "/sub/[redacted]", logredact.HTTPPathForLog("/sub/secret-token-here"))
	assert.Equal(t, "/sub/[redacted]", logredact.HTTPPathForLog("/sub/"))
	assert.Equal(t, "/api/v1/x", logredact.HTTPPathForLog("/api/v1/x"))
}

func TestProviderPaymentIDForLog(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", logredact.ProviderPaymentIDForLog("  "))
	assert.Equal(t, "[redacted:len=8]", logredact.ProviderPaymentIDForLog("12345678"))
	assert.Equal(t, "[redacted:len=12:suffix=90ab]", logredact.ProviderPaymentIDForLog("deadbeef90ab"))
}
