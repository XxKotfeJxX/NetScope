package reports

import (
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
)

type Comment struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
	RunID       uuid.UUID `json:"runId"`
	AuthorID    uuid.UUID `json:"authorId"`
	AuthorName  string    `json:"authorName"`
	AuthorEmail string    `json:"authorEmail"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PublicLink struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspaceId"`
	RunID        uuid.UUID  `json:"runId"`
	TokenPrefix  string     `json:"tokenPrefix"`
	CreatedBy    uuid.UUID  `json:"createdBy"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	LastViewedAt *time.Time `json:"lastViewedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	TokenHash    []byte     `json:"-"`
}

type CreatedPublicLink struct {
	PublicLink
	Token string `json:"token"`
}

type PublicReport struct {
	WorkspaceName string                    `json:"workspaceName"`
	PublishedAt   time.Time                 `json:"publishedAt"`
	ExpiresAt     time.Time                 `json:"expiresAt"`
	Run           diagnostics.DiagnosticRun `json:"run"`
}
