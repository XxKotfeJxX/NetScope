package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/google/uuid"
)

type apiMonitoringRepository struct {
	monitoring.Repository
	targets map[uuid.UUID]monitoring.Target
}

func (r *apiMonitoringRepository) CreateTarget(
	_ context.Context,
	target monitoring.Target,
) error {
	r.targets[target.ID] = target
	return nil
}

func (r *apiMonitoringRepository) GetTarget(
	_ context.Context,
	id uuid.UUID,
) (monitoring.Target, error) {
	target, exists := r.targets[id]
	if !exists {
		return monitoring.Target{}, monitoring.ErrTargetNotFound
	}
	return target, nil
}

func (r *apiMonitoringRepository) ListTargets(
	context.Context,
	uuid.UUID,
	int,
	int,
) (monitoring.Page, error) {
	items := make([]monitoring.Target, 0, len(r.targets))
	for _, target := range r.targets {
		items = append(items, target)
	}
	return monitoring.Page{
		Items: items, Page: 1, PageSize: 20, TotalItems: int64(len(items)),
		TotalPages: 1,
	}, nil
}

type apiRunService struct{}

func (apiRunService) Supports(diagnostics.CheckType) bool {
	return true
}

func (apiRunService) Create(
	context.Context,
	string,
	[]diagnostics.CheckType,
	diagnostics.RunOptions,
) (diagnostics.DiagnosticRun, error) {
	return diagnostics.DiagnosticRun{}, nil
}

func (apiRunService) CreateInWorkspace(
	context.Context,
	uuid.UUID,
	string,
	[]diagnostics.CheckType,
	diagnostics.RunOptions,
) (diagnostics.DiagnosticRun, error) {
	return diagnostics.DiagnosticRun{}, nil
}

func (apiRunService) Get(
	context.Context,
	uuid.UUID,
) (diagnostics.DiagnosticRun, error) {
	return diagnostics.DiagnosticRun{}, nil
}

func TestMonitoringTargetLifecycleAPI(t *testing.T) {
	t.Parallel()

	repository := &apiMonitoringRepository{
		targets: make(map[uuid.UUID]monitoring.Target),
	}
	service := monitoring.NewService(repository, apiRunService{})
	router := NewRouter(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Monitoring: service,
	})
	body := `{
		"name":"Production API",
		"address":"example.com",
		"tags":["production"],
		"checks":["dns"],
		"intervalSeconds":300,
		"failureThreshold":3,
		"options":{"timeoutMs":5000,"httpMethod":"GET","followRedirects":true,
			"maxRedirects":5,"ipVersion":"auto","pingPackets":4,"maxHops":20}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/targets", strings.NewReader(body))
	workspaceID := uuid.New()
	request = request.WithContext(identity.WithPrincipal(
		request.Context(),
		identity.Principal{
			Workspace: identity.Workspace{
				ID: workspaceID, Role: identity.RoleOperator,
			},
		},
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"name":"Production API"`) {
		t.Fatalf("create body = %s", response.Body)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil)
	request = request.WithContext(identity.WithPrincipal(
		request.Context(),
		identity.Principal{
			Workspace: identity.Workspace{
				ID: workspaceID, Role: identity.RoleOperator,
			},
		},
	))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"totalItems":1`) {
		t.Fatalf("list status = %d, body = %s", response.Code, response.Body)
	}
}
