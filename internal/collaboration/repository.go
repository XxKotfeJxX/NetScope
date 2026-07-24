package collaboration

import (
	"context"
	"errors"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput  = errors.New("invalid collaboration input")
	ErrMemberExists  = errors.New("workspace member already exists")
	ErrMemberMissing = errors.New("workspace member not found")
	ErrLastOwner     = errors.New("workspace must retain at least one owner")
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
}
