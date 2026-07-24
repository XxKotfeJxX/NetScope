package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/XxKotfeJxX/netscope/internal/target"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	router := NewRouter(Dependencies{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:   "test",
		WebOrigin: "http://localhost:5173",
	})
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"version":"test"`) {
		t.Fatalf("response = %q, want version", response.Body.String())
	}
}

func TestAPIErrorDoesNotExposeInternalDetails(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()

	writeAPIError(response, request, target.ErrAddressBlocked)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"code":"target_blocked"`) {
		t.Fatalf("response = %q, want target_blocked", body)
	}
	if !strings.Contains(body, `"requestId":"request-123"`) {
		t.Fatalf("response = %q, want request ID", body)
	}
	if strings.Contains(body, target.ErrAddressBlocked.Error()) {
		t.Fatalf("response exposed internal error: %q", body)
	}
}
