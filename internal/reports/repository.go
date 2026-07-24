package reports

import (
	"context"
	"errors"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid report collaboration input")
	ErrCommentMissing    = errors.New("report comment not found")
	ErrPublicLinkMissing = errors.New("public report link not found")
)

type ResolvedPublicLink struct {
	PublicLink
	WorkspaceName string
}

type Repository interface {
	ListComments(context.Context, uuid.UUID, uuid.UUID) ([]Comment, error)
	Comment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Comment, error)
	CreateComment(context.Context, Comment, collaboration.AuditEvent) error
	DeleteComment(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
		collaboration.AuditEvent,
	) error
	ListPublicLinks(context.Context, uuid.UUID, uuid.UUID) ([]PublicLink, error)
	CreatePublicLink(context.Context, PublicLink, collaboration.AuditEvent) error
	RevokePublicLink(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
		time.Time,
		collaboration.AuditEvent,
	) error
	ResolvePublicLink(context.Context, []byte, time.Time) (ResolvedPublicLink, error)
}
