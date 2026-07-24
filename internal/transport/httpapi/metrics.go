package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"
)

type Metrics struct {
	requests      atomic.Uint64
	serverErrors  atomic.Uint64
	panics        atomic.Uint64
	inFlight      atomic.Int64
	durationNanos atomic.Uint64
}

func (m *Metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		m.requests.Add(1)
		m.inFlight.Add(1)
		defer func() {
			m.inFlight.Add(-1)
			m.durationNanos.Add(uint64(time.Since(started)))
		}()

		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if recorder.status >= http.StatusInternalServerError {
			m.serverErrors.Add(1)
		}
	})
}

func (m *Metrics) serveHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(
		w,
		"# HELP netscope_http_requests_total Total HTTP requests handled.\n"+
			"# TYPE netscope_http_requests_total counter\n"+
			"netscope_http_requests_total %d\n"+
			"# HELP netscope_http_server_errors_total Total HTTP 5xx responses.\n"+
			"# TYPE netscope_http_server_errors_total counter\n"+
			"netscope_http_server_errors_total %d\n"+
			"# HELP netscope_http_panics_total Total recovered HTTP panics.\n"+
			"# TYPE netscope_http_panics_total counter\n"+
			"netscope_http_panics_total %d\n"+
			"# HELP netscope_http_requests_in_flight Current HTTP requests.\n"+
			"# TYPE netscope_http_requests_in_flight gauge\n"+
			"netscope_http_requests_in_flight %d\n"+
			"# HELP netscope_http_request_duration_seconds_total Cumulative request duration.\n"+
			"# TYPE netscope_http_request_duration_seconds_total counter\n"+
			"netscope_http_request_duration_seconds_total %.6f\n",
		m.requests.Load(),
		m.serverErrors.Load(),
		m.panics.Load(),
		m.inFlight.Load(),
		float64(m.durationNanos.Load())/float64(time.Second),
	)
}

func recoverPanics(
	logger *slog.Logger,
	metrics *Metrics,
) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					metrics.panics.Add(1)
					logger.Error(
						"recovered HTTP panic",
						"request_id", r.Header.Get("X-Request-ID"),
						"method", r.Method,
						"path", safeRequestPath(r.URL.Path),
						"panic", fmt.Sprint(recovered),
						"stack", string(debug.Stack()),
					)
					writeJSON(w, http.StatusInternalServerError, errorEnvelope{
						Error: apiError{
							Code:      "internal_error",
							Message:   "An unexpected error occurred.",
							RequestID: r.Header.Get("X-Request-ID"),
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
