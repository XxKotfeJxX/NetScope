package httpapi

import (
	"errors"
	"net/http"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

var (
	errInvalidRequest    = errors.New("invalid request")
	errUnsupportedFormat = errors.New("unsupported export format")
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"requestId"`
}

func writeAPIError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "An unexpected error occurred."
	var details map[string]any

	switch {
	case errors.Is(err, errInvalidRequest):
		status = http.StatusBadRequest
		code = "invalid_request"
		message = "The request body is invalid."
	case errors.Is(err, errUnsupportedFormat):
		status = http.StatusBadRequest
		code = "unsupported_format"
		message = "Only JSON and CSV exports are available."
	case errors.Is(err, target.ErrEmpty),
		errors.Is(err, target.ErrInvalid),
		errors.Is(err, target.ErrCIDR),
		errors.Is(err, target.ErrWildcard),
		errors.Is(err, target.ErrCredentials):
		status = http.StatusBadRequest
		code = "invalid_target"
		message = err.Error()
		details = map[string]any{"field": "target"}
	case errors.Is(err, target.ErrPort):
		status = http.StatusBadRequest
		code = "invalid_port"
		message = err.Error()
	case errors.Is(err, target.ErrAddressBlocked):
		status = http.StatusForbidden
		code = "target_blocked"
		message = "The target is blocked by the active network policy."
	case errors.Is(err, diagnostics.ErrUnsupportedCheck):
		status = http.StatusBadRequest
		code = "unsupported_check"
		message = "One or more requested checks are unavailable."
	case errors.Is(err, diagnostics.ErrInvalidOptions):
		status = http.StatusBadRequest
		code = "invalid_options"
		message = err.Error()
	case errors.Is(err, diagnostics.ErrRunNotFound):
		status = http.StatusNotFound
		code = "run_not_found"
		message = "The diagnostic run was not found."
	case errors.Is(err, diagnostics.ErrRunAlreadyFinished):
		status = http.StatusConflict
		code = "run_already_finished"
		message = "The diagnostic run has already finished."
	case errors.Is(err, diagnostics.ErrQueueFull):
		status = http.StatusServiceUnavailable
		code = "run_queue_full"
		message = "The diagnostic queue is full. Try again shortly."
	case errors.Is(err, monitoring.ErrInvalidTarget):
		status = http.StatusBadRequest
		code = "invalid_monitored_target"
		message = err.Error()
	case errors.Is(err, monitoring.ErrTargetNotFound):
		status = http.StatusNotFound
		code = "monitored_target_not_found"
		message = "The monitored target or nested resource was not found."
	}

	writeJSON(w, status, errorEnvelope{Error: apiError{
		Code: code, Message: message, Details: details, RequestID: r.Header.Get("X-Request-ID"),
	}})
}
