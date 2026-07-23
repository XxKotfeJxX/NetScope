package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
