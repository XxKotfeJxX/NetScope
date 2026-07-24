package diagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

func TestNormalizePorts(t *testing.T) {
	t.Parallel()

	got, err := normalizePorts([]int{443, 80, 443})
	if err != nil {
		t.Fatalf("normalizePorts() error = %v", err)
	}
	if want := []int{80, 443}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePorts() = %v, want %v", got, want)
	}
}

func TestNormalizePortsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	_, err := normalizePorts([]int{0})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("normalizePorts() error = %v, want ErrInvalidOptions", err)
	}
}

type probeForType struct {
	check CheckType
}

func (p probeForType) Type() CheckType {
	return p.check
}

func (p probeForType) Run(context.Context, target.Target, RunOptions) CheckResult {
	now := time.Now().UTC()
	return CheckResult{
		ID: uuid.New(), Type: p.check, Status: CheckPassed,
		Data: json.RawMessage(`{}`), StartedAt: now, CompletedAt: now,
	}
}

func TestCreateAppliesDeepDiagnosticDefaults(t *testing.T) {
	t.Parallel()

	repository := &memoryRepository{}
	manager := NewManager(
		repository,
		NewHub(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1,
		1,
		1,
		probeForType{check: CheckPing},
	)
	service := NewService(repository, manager, target.Policy{}, time.Second, 30*time.Second)

	workspaceID := uuid.New()
	run, err := service.Create(
		identity.WithPrincipal(context.Background(), identity.Principal{
			Workspace: identity.Workspace{ID: workspaceID, Role: identity.RoleOperator},
		}),
		"192.0.2.1",
		[]CheckType{CheckPing},
		RunOptions{TimeoutMS: 1000},
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if run.Options.PingPackets != 4 || run.Options.MaxHops != 20 {
		t.Fatalf("deep diagnostic defaults = %+v", run.Options)
	}
	if run.WorkspaceID != workspaceID {
		t.Fatalf("workspace = %s, want %s", run.WorkspaceID, workspaceID)
	}
	if _, err := service.Get(identity.WithPrincipal(
		context.Background(),
		identity.Principal{
			Workspace: identity.Workspace{
				ID: uuid.New(), Role: identity.RoleOperator,
			},
		},
	), run.ID); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("Get(other workspace) error = %v", err)
	}
}

func TestCreateRejectsUnavailableCheck(t *testing.T) {
	t.Parallel()

	repository := &memoryRepository{}
	manager := NewManager(
		repository,
		NewHub(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1,
		1,
		1,
		probeForType{check: CheckDNS},
	)
	service := NewService(repository, manager, target.Policy{}, time.Second, 30*time.Second)

	_, err := service.Create(
		identity.WithPrincipal(context.Background(), identity.Principal{
			Workspace: identity.Workspace{ID: uuid.New(), Role: identity.RoleOperator},
		}),
		"192.0.2.1",
		[]CheckType{CheckPing},
		RunOptions{TimeoutMS: 1000},
	)
	if !errors.Is(err, ErrUnsupportedCheck) {
		t.Fatalf("Create() error = %v, want ErrUnsupportedCheck", err)
	}
}
