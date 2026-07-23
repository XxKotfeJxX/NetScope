package diagnostics

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrRunNotFound = errors.New("diagnostic run not found")

type RunRepository interface {
	Create(context.Context, DiagnosticRun) error
	GetByID(context.Context, uuid.UUID) (DiagnosticRun, error)
	List(context.Context, ListFilter) (Page, error)
	UpdateStatus(context.Context, uuid.UUID, RunStatus) error
	SaveResult(context.Context, CheckResult) error
}
