package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(2, time.Minute, false)
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for attempt := 1; attempt <= 3; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		want := http.StatusNoContent
		if attempt == 3 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, want)
		}
		if response.Header().Get("RateLimit-Limit") != "2" {
			t.Fatalf("attempt %d omitted rate limit headers", attempt)
		}
		if attempt == 3 && response.Header().Get("Retry-After") == "" {
			t.Fatal("limited response omitted Retry-After")
		}
	}
}

func TestRateLimiterIgnoresForwardedAddressByDefault(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(1, time.Minute, false)
	handler := limiter.middleware(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for attempt, forwarded := range []string{"198.51.100.1", "198.51.100.2"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		request.Header.Set("X-Forwarded-For", forwarded)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		want := http.StatusNoContent
		if attempt == 1 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, response.Code, want)
		}
	}
}

func TestClientIPUsesForwardedAddressOnlyWhenTrusted(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.8, 203.0.113.2")

	if got := clientIP(request, false); got != "192.0.2.1" {
		t.Fatalf("untrusted clientIP = %q", got)
	}
	if got := clientIP(request, true); got != "198.51.100.8" {
		t.Fatalf("trusted clientIP = %q", got)
	}
}

func TestRateLimiterResetsExpiredWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	limiter := newRateLimiter(1, time.Minute, false)
	limiter.now = func() time.Time { return now }
	handler := limiter.middleware(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	handler.ServeHTTP(httptest.NewRecorder(), request)
	now = now.Add(time.Minute)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status after reset = %d", response.Code)
	}
}
