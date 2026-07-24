package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := securityHeaders(true)(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	for _, header := range []string{
		"Content-Security-Policy",
		"Permissions-Policy",
		"Referrer-Policy",
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"X-Frame-Options",
	} {
		if response.Header().Get(header) == "" {
			t.Errorf("%s header is missing", header)
		}
	}
}

func TestCSRFOriginGuardRejectsForeignCookieRequest(t *testing.T) {
	t.Parallel()

	handler := csrfOriginGuard(true, "https://netscope.example")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session"})
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Body.String(), `"code":"origin_rejected"`) {
		t.Fatalf("response = %q", response.Body.String())
	}
}

func TestCSRFOriginGuardAllowsConfiguredOriginAndBearer(t *testing.T) {
	t.Parallel()

	handler := csrfOriginGuard(true, "https://netscope.example")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	tests := []struct {
		name   string
		origin string
		bearer bool
	}{
		{name: "matching origin", origin: "https://netscope.example"},
		{name: "bearer authentication", origin: "https://attacker.example", bearer: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session"})
			request.Header.Set("Origin", test.origin)
			if test.bearer {
				request.Header.Set("Authorization", "Bearer ns_key_test")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
			}
		})
	}
}

func TestCORSRejectsForeignPreflight(t *testing.T) {
	t.Parallel()

	handler := cors("https://netscope.example")(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/runs", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("foreign origin received CORS allow header")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	t.Parallel()

	router := NewRouter(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:        "test",
		WebOrigin:      "http://localhost:5173",
		MetricsEnabled: true,
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "netscope_http_requests_total 1") {
		t.Fatalf("metrics = %q", response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("metrics response is cacheable")
	}
}
