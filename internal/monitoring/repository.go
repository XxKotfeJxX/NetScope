package monitoring

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrTargetNotFound = errors.New("monitored target not found")
	ErrInvalidTarget  = errors.New("invalid monitored target")
)

type Repository interface {
	CreateTarget(context.Context, Target) error
	GetTarget(context.Context, uuid.UUID) (Target, error)
	ListTargets(context.Context, uuid.UUID, int, int) (Page, error)
	UpdateTarget(context.Context, Target) error
	DeleteTarget(context.Context, uuid.UUID) error
	SetTargetEnabled(context.Context, uuid.UUID, bool) error
	ListChecks(context.Context, uuid.UUID, int, int) (CheckPage, error)
	CreateMaintenanceWindow(context.Context, MaintenanceWindow) error
	ListMaintenanceWindows(context.Context, uuid.UUID) ([]MaintenanceWindow, error)
	DeleteMaintenanceWindow(context.Context, uuid.UUID, uuid.UUID) error
	CreateNotificationChannel(context.Context, NotificationChannel) error
	ListNotificationChannels(context.Context, uuid.UUID) ([]NotificationChannel, error)
	DeleteNotificationChannel(context.Context, uuid.UUID, uuid.UUID) error
	ClaimDueTargets(context.Context, int) ([]Target, error)
	CreateScheduledCheck(context.Context, uuid.UUID, uuid.UUID) error
	ListPendingChecks(context.Context, int) ([]Check, error)
	CompleteCheck(context.Context, Check) (Target, bool, error)
	RecordDispatchFailure(context.Context, uuid.UUID, string) (Target, bool, error)
	Overview(context.Context, uuid.UUID, int) (Overview, error)
}
