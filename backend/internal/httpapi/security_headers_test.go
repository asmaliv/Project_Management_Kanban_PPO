package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders_Present(t *testing.T) {
	router := NewRouter(RouterParams{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                    "DENY",
		"Referrer-Policy":                    "no-referrer",
		"Strict-Transport-Security":          "max-age=63072000; includeSubDomains; preload",
		"Permissions-Policy":                "camera=(), microphone=(), geolocation=(), payment=()",
		"X-Permitted-Cross-Domain-Policies": "none",
	}

	for header, expectedVal := range expectedHeaders {
		val := rec.Header().Get(header)
		if val != expectedVal {
			t.Errorf("header %s: expected %q, got %q", header, expectedVal, val)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Errorf("Content-Security-Policy header is missing")
	}
}

func TestRateLimitGeneralWrite(t *testing.T) {
	handler := RateLimitGeneralWrite()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// GET requests should pass through without rate limiting
	for i := 0; i < 150; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET request %d should not be rate-limited, got status %d", i, rec.Code)
		}
	}

	// Write requests: first 120 should succeed
	for i := 0; i < 120; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil)
		req.RemoteAddr = "192.168.1.200:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("write request %d should succeed, got status %d", i, rec.Code)
		}
	}

	// 121st request should be rejected with 429 RATE_LIMITED
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil)
	req.RemoteAddr = "192.168.1.200:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("121st request should be 429, got %d", rec.Code)
	}

	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("expected Retry-After header 60, got %s", rec.Header().Get("Retry-After"))
	}
}

func TestRouter_NilLogger_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("router panicked with nil logger: %v", r)
		}
	}()

	router := NewRouter(RouterParams{
		Logger: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
