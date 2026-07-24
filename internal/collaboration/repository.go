package collaboration

import (
	"context"
	"errors"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput  = errors.New("invalid collaboration input")
	ErrMemberExists  = errors.New("workspace member already exists")
	ErrMemberMissing = errors.New("workspace member not found")
	ErrLastOwner     = errors.New("workspace must retain at least one owner")
	ErrAPIKeyMissing = errors.New("API key not found")
)

type Repository interface {
	ListMembers(context.Context, uuid.UUID) ([]Member, error)
	Member(context.Context, uuid.UUID, uuid.UUID) (Member, error)
	AddMember(
		context.Context,
		uuid.UUID,
		string,
		identity.Role,
		AuditEvent,
	) (Member, error)
	UpdateMemberRole(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		identity.Role,
		AuditEvent,
	) (Member, error)
	RemoveMember(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		AuditEvent,
	) error
	ListAudit(context.Context, uuid.UUID, int, int) ([]AuditEvent, int, error)
	ListAPIKeys(context.Context, uuid.UUID) ([]APIKey, error)
	CreateAPIKey(context.Context, APIKey, AuditEvent) error
	RevokeAPIKey(context.Context, uuid.UUID, uuid.UUID, time.Time, AuditEvent) error
}
