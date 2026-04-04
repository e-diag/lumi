package web

import (
	"net/http/httptest"
	"testing"
)

func TestSessionToken_RoundTrip(t *testing.T) {
	session, csrf, err := newSessionToken("secret")
	if err != nil {
		t.Fatalf("newSessionToken error: %v", err)
	}
	if csrf == "" {
		t.Fatal("csrf token is empty")
	}
	if !validateSessionToken(session, "secret") {
		t.Fatal("session should be valid")
	}
	if validateSessionToken(session, "wrong") {
		t.Fatal("session should be invalid with wrong secret")
	}
}

func TestSameOrigin(t *testing.T) {
	r := httptest.NewRequest("POST", "http://example.com/admin/api/users", nil)
	r.Host = "example.com"
	r.Header.Set("Origin", "http://example.com")
	if !sameOrigin(r) {
		t.Fatal("expected same origin true")
	}
	r.Header.Set("Origin", "http://evil.com")
	if sameOrigin(r) {
		t.Fatal("expected same origin false")
	}
}
