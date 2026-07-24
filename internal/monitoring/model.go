package monitoring

import (
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
)

type TargetStatus string

const (
	StatusPending     TargetStatus = "pending"
	StatusOperational TargetStatus = "operational"
	StatusWarning     TargetStatus = "warning"
	StatusUnavailable TargetStatus = "unavailable"
	StatusPaused      TargetStatus = "paused"
	StatusMaintenance TargetStatus = "maintenance"
)

type Target struct {
	ID                  uuid.UUID               `json:"id"`
	Name                string                  `json:"name"`
	Address             string                  `json:"address"`
	Tags                []string                `json:"tags"`
	Checks              []diagnostics.CheckType `json:"checks"`
	Options             diagnostics.RunOptions  `json:"options"`
	IntervalSeconds     int                     `json:"intervalSeconds"`
	Enabled             bool                    `json:"enabled"`
	FailureThreshold    int                     `json:"failureThreshold"`
	ConsecutiveFailures int                     `json:"consecutiveFailures"`
	Status              TargetStatus            `json:"status"`
	LastCheckedAt       *time.Time              `json:"lastCheckedAt,omitempty"`
	LastLatencyMS       *int64                  `json:"lastLatencyMs,omitempty"`
	TLSExpiresAt        *time.Time              `json:"tlsExpiresAt,omitempty"`
	NextCheckAt         time.Time               `json:"nextCheckAt"`
	CreatedAt           time.Time               `json:"createdAt"`
	UpdatedAt           time.Time               `json:"updatedAt"`
}

type TargetInput struct {
	Name             string                  `json:"name"`
	Address          string                  `json:"address"`
	Tags             []string                `json:"tags"`
	Checks           []diagnostics.CheckType `json:"checks"`
	Options          diagnostics.RunOptions  `json:"options"`
	IntervalSeconds  int                     `json:"intervalSeconds"`
	FailureThreshold int                     `json:"failureThreshold"`
}

type Check struct {
	ID           uuid.UUID    `json:"id"`
	TargetID     uuid.UUID    `json:"targetId"`
	RunID        *uuid.UUID   `json:"runId,omitempty"`
	Status       TargetStatus `json:"status"`
	LatencyMS    *int64       `json:"latencyMs,omitempty"`
	TLSExpiresAt *time.Time   `json:"tlsExpiresAt,omitempty"`
	ErrorMessage string       `json:"errorMessage,omitempty"`
	CheckedAt    *time.Time   `json:"checkedAt,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
}

type MaintenanceWindow struct {
	ID        uuid.UUID `json:"id"`
	TargetID  uuid.UUID `json:"targetId"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationKind string

const (
	NotificationEmail   NotificationKind = "email"
	NotificationWebhook NotificationKind = "webhook"
)

type NotificationChannel struct {
	ID          uuid.UUID        `json:"id"`
	TargetID    uuid.UUID        `json:"targetId"`
	Kind        NotificationKind `json:"kind"`
	Destination string           `json:"destination"`
	Enabled     bool             `json:"enabled"`
	CreatedAt   time.Time        `json:"createdAt"`
}

type Page struct {
	Items      []Target `json:"items"`
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`
	TotalItems int64    `json:"totalItems"`
	TotalPages int      `json:"totalPages"`
}

type CheckPage struct {
	Items      []Check `json:"items"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalItems int64   `json:"totalItems"`
	TotalPages int     `json:"totalPages"`
}
