package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowMessage_WithinLimit(t *testing.T) {
	rl := New(10.0, 5, 10.0/60.0, 2, 5)
	defer rl.Stop()

	chatID := "chat-1@s.whatsapp.net"

	if !rl.AllowMessage(chatID) {
		t.Fatal("first message should be allowed")
	}
	if !rl.AllowMessage(chatID) {
		t.Fatal("second message should be allowed (within burst=2)")
	}
	if rl.AllowMessage(chatID) {
		t.Fatal("third message should be rejected (burst=2 exhausted)")
	}
}

func TestAllowMessage_DifferentChats(t *testing.T) {
	rl := New(10.0, 5, 10.0/60.0, 1, 5)
	defer rl.Stop()

	chatA := "chatA@s.whatsapp.net"
	chatB := "chatB@s.whatsapp.net"

	if !rl.AllowMessage(chatA) {
		t.Fatal("chatA first message should be allowed")
	}
	// chatA burst exhausted
	if rl.AllowMessage(chatA) {
		t.Fatal("chatA second message should be rejected (burst=1)")
	}

	// chatB should still have its own independent limit
	if !rl.AllowMessage(chatB) {
		t.Fatal("chatB first message should be allowed (independent limiter)")
	}
}

func TestHTTPMiddleware_NormalRequest(t *testing.T) {
	rl := New(100.0, 50, 10.0, 5, 60)
	defer rl.Stop()

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", rec.Body.String())
	}
}

func TestHTTPMiddleware_HealthExempt(t *testing.T) {
	// Very restrictive rate limit
	rl := New(0.001, 1, 10.0, 5, 1)
	defer rl.Stop()

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the API rate limit first
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// /health should still pass even after limit exceeded
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("health request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestHTTPMiddleware_RateLimited(t *testing.T) {
	// apiRate=0.001 (basically never refills), apiBurst=1
	rl := New(0.001, 1, 10.0, 5, 60)
	defer rl.Stop()

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	remoteAddr := "192.168.1.1:9999"

	// First request should pass (burst=1)
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec.Code)
	}

	// Second request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = remoteAddr
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec.Code)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
}

func TestHTTPMiddleware_LoginStricter(t *testing.T) {
	// loginPerMin=1, so only 1 login attempt allowed; apiRate generous
	rl := New(100.0, 50, 10.0, 5, 1)
	defer rl.Stop()

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	remoteAddr := "10.10.10.10:5555"

	// First login request should pass
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first login: expected 200, got %d", rec.Code)
	}

	// Second login request should be rate limited (loginPerMin=1)
	req = httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = remoteAddr
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second login: expected 429, got %d", rec.Code)
	}

	// But a normal API request from the same IP should still pass
	req = httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = remoteAddr
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("normal API after login limit: expected 200, got %d", rec.Code)
	}
}

func TestStop(t *testing.T) {
	rl := New(10.0, 5, 10.0, 5, 60)

	// Stop should not panic
	rl.Stop()
}
