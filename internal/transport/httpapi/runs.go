package httpapi

import (
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
	type capability struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason,omitempty"`
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version": h.version,
		"checks": map[string]capability{
			"dns":        {Available: true},
			"tcp":        {Available: false, Reason: "not_implemented"},
			"http":       {Available: false, Reason: "not_implemented"},
			"tls":        {Available: false, Reason: "not_implemented"},
			"ping":       {Available: false, Reason: "not_implemented"},
			"traceroute": {Available: false, Reason: "not_implemented"},
		},
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
	if format := r.URL.Query().Get("format"); format != "" && format != "json" {
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
	w.Header().Set("Content-Disposition", `attachment; filename="netscope-`+id.String()+`.json"`)
	writeJSON(w, http.StatusOK, run)
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
