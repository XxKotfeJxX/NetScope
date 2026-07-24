package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
)

func (h apiHandler) runEvents(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r)
	if err != nil {
		writeAPIError(w, r, diagnostics.ErrRunNotFound)
		return
	}
	if _, err := h.runs.Get(r.Context(), id); err != nil {
		writeAPIError(w, r, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, r, fmt.Errorf("streaming is unavailable"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	events, unsubscribe := h.events.Subscribe(id)
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, open := <-events:
			if !open {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
