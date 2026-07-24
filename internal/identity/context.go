package identity

import (
	"context"

	"github.com/google/uuid"
)

type principalContextKey struct{}
type systemContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

func WithSystemAccess(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemContextKey{}, true)
}

func HasSystemAccess(ctx context.Context) bool {
	allowed, _ := ctx.Value(systemContextKey{}).(bool)
	return allowed
}

func WorkspaceID(ctx context.Context) (uuid.UUID, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.Workspace.ID == uuid.Nil {
		return uuid.Nil, ErrUnauthenticated
	}
	return principal.Workspace.ID, nil
}

func AuthorizeWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	if HasSystemAccess(ctx) {
		return nil
	}
	active, err := WorkspaceID(ctx)
	if err != nil {
		return err
	}
	if active != workspaceID {
		return ErrForbidden
	}
	return nil
}

func RoleAtLeast(actual Role, required Role) bool {
	return roleRank(actual) >= roleRank(required)
}

func roleRank(role Role) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleOperator:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}
