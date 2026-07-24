package monitoring

import (
	"context"
	"testing"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
)

type memoryRepository struct {
	Repository
	target Target
}

func (r *memoryRepository) CreateTarget(_ context.Context, target Target) error {
	r.target = target
	return nil
}

func (r *memoryRepository) GetTarget(
	_ context.Context,
	id uuid.UUID,
) (Target, error) {
	if r.target.ID != id {
		return Target{}, ErrTargetNotFound
	}
	return r.target, nil
}

type availableRuns struct{}

func (availableRuns) Supports(check diagnostics.CheckType) bool {
	return check == diagnostics.CheckDNS || check == diagnostics.CheckHTTP
}

func TestCreateTargetNormalizesDefaultsAndTags(t *testing.T) {
	t.Parallel()

	repository := &memoryRepository{}
	service := NewService(repository, availableRuns{})
	created, err := service.CreateTarget(context.Background(), TargetInput{
		Name:    "  Production API  ",
		Address: "HTTPS://Example.COM/path",
		Tags:    []string{"Production", " api ", "production"},
		Checks:  []diagnostics.CheckType{diagnostics.CheckDNS, diagnostics.CheckHTTP},
	})
	if err != nil {
		t.Fatalf("CreateTarget() error = %v", err)
	}
	if created.Name != "Production API" || created.Address != "HTTPS://Example.COM/path" {
		t.Fatalf("created target = %+v", created)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "api" ||
		created.Tags[1] != "production" {
		t.Fatalf("tags = %#v", created.Tags)
	}
	if created.IntervalSeconds != 300 || created.FailureThreshold != 3 {
		t.Fatalf("defaults = interval %d, threshold %d",
			created.IntervalSeconds, created.FailureThreshold)
	}
	if created.Options.TimeoutMS != 5000 ||
		created.Options.HTTPMethod != "GET" ||
		created.Options.IPVersion != "auto" {
		t.Fatalf("options = %+v", created.Options)
	}
	if !created.Enabled || created.Status != StatusPending {
		t.Fatalf("state = enabled %t, status %s", created.Enabled, created.Status)
	}
}

func TestCreateTargetRejectsUnavailableCheck(t *testing.T) {
	t.Parallel()

	service := NewService(&memoryRepository{}, availableRuns{})
	_, err := service.CreateTarget(context.Background(), TargetInput{
		Name: "Router", Address: "1.1.1.1",
		Checks: []diagnostics.CheckType{diagnostics.CheckTraceroute},
	})
	if err == nil {
		t.Fatal("CreateTarget() error = nil, want unavailable check error")
	}
}
