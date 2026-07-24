package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestWorkspaceAuthorization(t *testing.T) {
	t.Parallel()

	workspaceID := uuid.New()
	ctx := WithPrincipal(context.Background(), Principal{
		Workspace: Workspace{ID: workspaceID, Role: RoleOperator},
	})
	if err := AuthorizeWorkspace(ctx, workspaceID); err != nil {
		t.Fatalf("AuthorizeWorkspace(own) error = %v", err)
	}
	if err := AuthorizeWorkspace(ctx, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("AuthorizeWorkspace(other) error = %v", err)
	}
	if err := AuthorizeWorkspace(
		WithSystemAccess(context.Background()),
		uuid.New(),
	); err != nil {
		t.Fatalf("AuthorizeWorkspace(system) error = %v", err)
	}
}

func TestRoleHierarchy(t *testing.T) {
	t.Parallel()

	if !RoleAtLeast(RoleOwner, RoleAdmin) ||
		!RoleAtLeast(RoleOperator, RoleViewer) ||
		RoleAtLeast(RoleViewer, RoleOperator) {
		t.Fatal("role hierarchy is incorrect")
	}
}
