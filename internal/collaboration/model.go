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

type APIKey struct {
	ID          uuid.UUID     `json:"id"`
	WorkspaceID uuid.UUID     `json:"workspaceId"`
	Name        string        `json:"name"`
	Prefix      string        `json:"prefix"`
	Role        identity.Role `json:"role"`
	CreatedBy   uuid.UUID     `json:"createdBy"`
	ExpiresAt   time.Time     `json:"expiresAt"`
	LastUsedAt  *time.Time    `json:"lastUsedAt,omitempty"`
	RevokedAt   *time.Time    `json:"revokedAt,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	TokenHash   []byte        `json:"-"`
}

type CreateAPIKeyInput struct {
	Name      string        `json:"name"`
	Role      identity.Role `json:"role"`
	ExpiresAt *time.Time    `json:"expiresAt,omitempty"`
}

type CreatedAPIKey struct {
	APIKey
	Token string `json:"token"`
}
