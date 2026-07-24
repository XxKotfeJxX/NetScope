package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid identity input")
	ErrEmailExists       = errors.New("email is already registered")
	ErrUnauthenticated   = errors.New("authentication required")
	ErrForbidden         = errors.New("insufficient workspace permission")
	ErrUserNotFound      = errors.New("user not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

type Repository interface {
	CreateRegistration(
		context.Context,
		User,
		string,
		Workspace,
		Membership,
		Session,
	) error
	UserByEmail(context.Context, string) (User, string, error)
	CreateSession(context.Context, Session) error
	SessionByTokenHash(context.Context, []byte) (Session, User, error)
	APIKeyByTokenHash(context.Context, []byte) (APIKeyCredential, error)
	DeleteSession(context.Context, []byte) error
	ListWorkspaces(context.Context, uuid.UUID) ([]Workspace, error)
	CreateWorkspace(context.Context, Workspace, Membership) error
	Membership(context.Context, uuid.UUID, uuid.UUID) (Membership, error)
}
