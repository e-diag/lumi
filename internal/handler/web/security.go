package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func newSessionToken(secret string) (session string, csrf string, err error) {
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return "", "", fmt.Errorf("read nonce: %w", err)
	}
	csrfBytes := make([]byte, 16)
	if _, err = rand.Read(csrfBytes); err != nil {
		return "", "", fmt.Errorf("read csrf: %w", err)
	}
	csrf = hex.EncodeToString(csrfBytes)
	exp := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf("%d.%s", exp, base64.RawURLEncoding.EncodeToString(nonce))
	sig := sign(secret, payload)
	return payload + "." + sig, csrf, nil
}

func validateSessionToken(token, secret string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(sign(secret, payload)), []byte(parts[2])) {
		return false
	}
	exp, err := parseInt64(parts[0])
	if err != nil {
		return false
	}
	return time.Now().Unix() < exp
}

func sign(secret, payload string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(payload))
	return hex.EncodeToString(m.Sum(nil))
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://"+r.Host) || strings.HasPrefix(origin, "https://"+r.Host)
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid int")
		}
		n = n*10 + int64(ch-'0')
	}
	return n, nil
}
