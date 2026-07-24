package collaboration

import (
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
)

type Member struct {
	UserID      uuid.UUID     `json:"userId"`
	Email       string        `json:"email"`
	DisplayName string        `json:"displayName"`
	Role        identity.Role `json:"role"`
	JoinedAt    time.Time     `json:"joinedAt"`
}

type AuditEvent struct {
	ID           uuid.UUID      `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspaceId"`
	ActorUserID  uuid.UUID      `json:"actorUserId"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   *uuid.UUID     `json:"resourceId,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type AddMemberInput struct {
	Email string        `json:"email"`
	Role  identity.Role `json:"role"`
}

type AuditPage struct {
	Items      []AuditEvent `json:"items"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalItems int          `json:"totalItems"`
	TotalPages int          `json:"totalPages"`
}
