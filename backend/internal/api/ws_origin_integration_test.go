package api

import (
	"net/http"
	"testing"

	"xray2wg/backend/internal/security/origin"
)

func TestWebSocketCheckOriginAllowed(t *testing.T) {
	cfg, err := origin.NewConfig("https://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	check := func(r *http.Request) bool {
		return cfg.AllowOrigin(r.Header.Get("Origin"))
	}
	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Origin", "https://localhost:5173")
	if !check(req) {
		t.Fatal("expected allowed")
	}
}

func TestWebSocketCheckOriginDisallowed(t *testing.T) {
	cfg, err := origin.NewConfig("https://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	check := func(r *http.Request) bool {
		return cfg.AllowOrigin(r.Header.Get("Origin"))
	}
	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Origin", "https://attacker.example")
	if check(req) {
		t.Fatal("expected disallowed")
	}
}
