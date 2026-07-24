package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
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

func TestWorkspaceRoleGuardRejectsViewerWrites(t *testing.T) {
	t.Parallel()

	guard := identityHandler{}.requireRole(identity.RoleOperator)
	handler := guard(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/targets", nil)
	request = request.WithContext(identity.WithPrincipal(
		request.Context(),
		identity.Principal{
			Workspace: identity.Workspace{
				ID: uuid.New(), Role: identity.RoleViewer,
			},
		},
	))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCapabilitiesExposeRuntimeProbeAvailability(t *testing.T) {
	t.Parallel()

	router := NewRouter(Dependencies{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:   "0.2.0-test",
		WebOrigin: "http://localhost:5173",
		Checks: map[string]Capability{
			"dns":  {Available: true},
			"ping": {Available: false, Reason: "raw_icmp_unavailable"},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var payload struct {
		Version string                `json:"version"`
		Checks  map[string]Capability `json:"checks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if payload.Version != "0.2.0-test" || payload.Checks["ping"].Available {
		t.Fatalf("capabilities = %+v", payload)
	}
	if payload.Checks["ping"].Reason != "raw_icmp_unavailable" {
		t.Fatalf("ping reason = %q", payload.Checks["ping"].Reason)
	}
}

func TestSafeRequestPathRedactsPublicReportToken(t *testing.T) {
	t.Parallel()

	token := "ns_share_secret-that-must-never-reach-logs"
	path := safeRequestPath("/api/v1/public/reports/" + token)
	if path != "/api/v1/public/reports/[redacted]" {
		t.Fatalf("safeRequestPath() = %q", path)
	}
	if strings.Contains(path, token) {
		t.Fatal("safeRequestPath() retained the public report token")
	}
	if other := safeRequestPath("/api/v1/runs/123"); other != "/api/v1/runs/123" {
		t.Fatalf("safeRequestPath(ordinary) = %q", other)
	}
}
