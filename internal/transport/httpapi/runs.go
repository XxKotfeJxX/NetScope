package httpapi

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type apiHandler struct {
	runs    *diagnostics.Service
	events  diagnostics.EventPublisher
	version string
	runtime RuntimeInfo
	checks  map[string]Capability
}

type createRunRequest struct {
	Target  string                  `json:"target"`
	Checks  []diagnostics.CheckType `json:"checks"`
	Options diagnostics.RunOptions  `json:"options"`
}

type createRunResponse struct {
	ID        uuid.UUID             `json:"id"`
	Status    diagnostics.RunStatus `json:"status"`
	CreatedAt string                `json:"createdAt"`
}

func (h apiHandler) capabilities(w http.ResponseWriter, _ *http.Request) {
	checks := h.checks
	if checks == nil {
		checks = map[string]Capability{
			"dns": {Available: true}, "tcp": {Available: true},
			"http": {Available: true}, "tls": {Available: true},
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version": h.version,
		"checks":  checks,
		"runtime": h.runtime,
	})
}

func (h apiHandler) createRun(w http.ResponseWriter, r *http.Request) {
	var request createRunRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, r, errInvalidRequest)
		return
	}

	run, err := h.runs.Create(r.Context(), request.Target, request.Checks, request.Options)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/runs/"+run.ID.String())
	writeJSON(w, http.StatusCreated, createRunResponse{
		ID: run.ID, Status: run.Status, CreatedAt: run.CreatedAt.Format(time.RFC3339Nano),
	})
}

func (h apiHandler) getRun(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, diagnostics.ErrRunNotFound)
		return
	}
	run, err := h.runs.Get(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h apiHandler) listRuns(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)
	status := diagnostics.RunStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	result, err := h.runs.List(r.Context(), diagnostics.ListFilter{
		Page: page, PageSize: pageSize, Status: status,
	})
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h apiHandler) cancelRun(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, diagnostics.ErrRunNotFound)
		return
	}
	if err := h.runs.Cancel(r.Context(), id); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h apiHandler) exportRun(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format != "" && format != "json" && format != "csv" {
		writeAPIError(w, r, errUnsupportedFormat)
		return
	}
	id, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, diagnostics.ErrRunNotFound)
		return
	}
	run, err := h.runs.Get(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	if format == "csv" {
		payload, err := runCSV(run)
		if err != nil {
			writeAPIError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set(
			"Content-Disposition",
			`attachment; filename="netscope-`+id.String()+`.csv"`,
		)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		return
	}
	w.Header().Set(
		"Content-Disposition",
		`attachment; filename="netscope-`+id.String()+`.json"`,
	)
	writeJSON(w, http.StatusOK, run)
}

func runCSV(run diagnostics.DiagnosticRun) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	header := []string{
		"run_id", "target", "normalized_host", "normalized_url", "run_status",
		"requested_checks", "options_json", "created_at", "started_at", "completed_at",
		"check_type", "check_status", "duration_ms", "summary", "error_code",
		"error_message", "data_json",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	options, err := json.Marshal(run.Options)
	if err != nil {
		return nil, err
	}
	checks := make([]string, len(run.RequestedChecks))
	for index, check := range run.RequestedChecks {
		checks[index] = string(check)
	}
	base := []string{
		run.ID.String(), run.TargetInput, run.NormalizedHost, run.NormalizedURL,
		string(run.Status), strings.Join(checks, "|"), string(options),
		run.CreatedAt.Format(time.RFC3339Nano), formatOptionalTime(run.StartedAt),
		formatOptionalTime(run.CompletedAt),
	}

	if len(run.Results) == 0 {
		if err := writer.Write(safeCSVRow(append(base, make([]string, 7)...))); err != nil {
			return nil, err
		}
	} else {
		for _, result := range run.Results {
			row := append(append([]string{}, base...),
				string(result.Type), string(result.Status),
				strconv.FormatInt(result.DurationMS, 10), result.Summary,
				result.ErrorCode, result.ErrorMessage, string(result.Data),
			)
			if err := writer.Write(safeCSVRow(row)); err != nil {
				return nil, err
			}
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func safeCSVRow(row []string) []string {
	safe := make([]string, len(row))
	for index, value := range row {
		if strings.HasPrefix(value, "=") || strings.HasPrefix(value, "+") ||
			strings.HasPrefix(value, "-") || strings.HasPrefix(value, "@") ||
			strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") {
			value = "'" + value
		}
		safe[index] = value
	}
	return safe
}

func pathUUID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

func queryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
