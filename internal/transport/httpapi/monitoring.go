package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type maintenanceRequest struct {
	StartsAt time.Time `json:"startsAt"`
	EndsAt   time.Time `json:"endsAt"`
	Reason   string    `json:"reason"`
}

type notificationRequest struct {
	Kind        monitoring.NotificationKind `json:"kind"`
	Destination string                      `json:"destination"`
}

func (h apiHandler) createTarget(w http.ResponseWriter, r *http.Request) {
	var input monitoring.TargetInput
	if err := decodeRequest(w, r, &input); err != nil {
		writeAPIError(w, r, err)
		return
	}
	created, err := h.monitoring.CreateTarget(r.Context(), input)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/targets/"+created.ID.String())
	writeJSON(w, http.StatusCreated, created)
}

func (h apiHandler) listTargets(w http.ResponseWriter, r *http.Request) {
	page, err := h.monitoring.ListTargets(
		r.Context(),
		queryInt(r, "page", 1),
		queryInt(r, "pageSize", 20),
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h apiHandler) getTarget(w http.ResponseWriter, r *http.Request) {
	id, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	target, err := h.monitoring.GetTarget(r.Context(), id)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, target)
}

func (h apiHandler) updateTarget(w http.ResponseWriter, r *http.Request) {
	id, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	var input monitoring.TargetInput
	if err := decodeRequest(w, r, &input); err != nil {
		writeAPIError(w, r, err)
		return
	}
	updated, err := h.monitoring.UpdateTarget(r.Context(), id, input)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h apiHandler) deleteTarget(w http.ResponseWriter, r *http.Request) {
	id, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	if err := h.monitoring.DeleteTarget(r.Context(), id); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h apiHandler) pauseTarget(w http.ResponseWriter, r *http.Request) {
	h.setTargetEnabled(w, r, false)
}

func (h apiHandler) resumeTarget(w http.ResponseWriter, r *http.Request) {
	h.setTargetEnabled(w, r, true)
}

func (h apiHandler) setTargetEnabled(
	w http.ResponseWriter,
	r *http.Request,
	enabled bool,
) {
	id, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	if err := h.monitoring.SetTargetEnabled(r.Context(), id, enabled); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h apiHandler) listTargetChecks(w http.ResponseWriter, r *http.Request) {
	id, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	page, err := h.monitoring.ListChecks(
		r.Context(),
		id,
		queryInt(r, "page", 1),
		queryInt(r, "pageSize", 50),
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h apiHandler) createMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	var request maintenanceRequest
	if err := decodeRequest(w, r, &request); err != nil {
		writeAPIError(w, r, err)
		return
	}
	window, err := h.monitoring.CreateMaintenanceWindow(
		r.Context(),
		targetID,
		request.StartsAt,
		request.EndsAt,
		request.Reason,
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, window)
}

func (h apiHandler) listMaintenanceWindows(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	windows, err := h.monitoring.ListMaintenanceWindows(r.Context(), targetID)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, windows)
}

func (h apiHandler) deleteMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	windowID, err := monitoringPathUUID(r, "windowID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	if err := h.monitoring.DeleteMaintenanceWindow(
		r.Context(),
		targetID,
		windowID,
	); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h apiHandler) createNotificationChannel(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	var request notificationRequest
	if err := decodeRequest(w, r, &request); err != nil {
		writeAPIError(w, r, err)
		return
	}
	channel, err := h.monitoring.CreateNotificationChannel(
		r.Context(),
		targetID,
		request.Kind,
		request.Destination,
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, channel)
}

func (h apiHandler) listNotificationChannels(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	channels, err := h.monitoring.ListNotificationChannels(r.Context(), targetID)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

func (h apiHandler) deleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	targetID, err := monitoringPathUUID(r, "targetID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	channelID, err := monitoringPathUUID(r, "channelID")
	if err != nil {
		writeAPIError(w, r, monitoring.ErrTargetNotFound)
		return
	}
	if err := h.monitoring.DeleteNotificationChannel(
		r.Context(),
		targetID,
		channelID,
	); err != nil {
		writeAPIError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h apiHandler) monitoringOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.monitoring.Overview(
		r.Context(),
		queryInt(r, "limit", 50),
	)
	if err != nil {
		writeAPIError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalidRequest
	}
	return nil
}

func monitoringPathUUID(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}
