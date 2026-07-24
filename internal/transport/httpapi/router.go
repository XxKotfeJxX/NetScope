package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Logger    *slog.Logger
	Pool      *pgxpool.Pool
	Version   string
	WebOrigin string
	Runs      *diagnostics.Service
	Events    diagnostics.EventPublisher
	Runtime   RuntimeInfo
	Checks    map[string]Capability
}

type Capability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type RuntimeInfo struct {
	DefaultTimeoutMS int    `json:"defaultTimeoutMs"`
	MaxTimeoutMS     int    `json:"maxTimeoutMs"`
	RunWorkers       int    `json:"runWorkers"`
	ProbeConcurrency int    `json:"probeConcurrency"`
	NetworkPolicy    string `json:"networkPolicy"`
}

type healthHandler struct {
	pool    *pgxpool.Pool
	version string
}

func NewRouter(deps Dependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(requestID)
	router.Use(requestLogger(deps.Logger))
	router.Use(cors(deps.WebOrigin))

	health := healthHandler{pool: deps.Pool, version: deps.Version}
	router.Get("/healthz", health.live)
	router.Get("/readyz", health.ready)

	api := apiHandler{
		runs: deps.Runs, events: deps.Events, version: deps.Version,
		runtime: deps.Runtime, checks: deps.Checks,
	}
	limiter := newRateLimiter(10, time.Minute)
	router.Route("/api/v1", func(router chi.Router) {
		router.Get("/capabilities", api.capabilities)
		router.With(limiter.middleware).Post("/runs", api.createRun)
		router.Get("/runs", api.listRuns)
		router.Get("/runs/{id}", api.getRun)
		router.Post("/runs/{id}/cancel", api.cancelRun)
		router.Get("/runs/{id}/events", api.runEvents)
		router.Get("/runs/{id}/export", api.exportRun)
	})

	return router
}

func (h healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "netscope",
		"version": h.version,
	})
}

func (h healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.Info("request completed",
				"request_id", r.Header.Get("X-Request-ID"),
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(started).Milliseconds(),
			)
		})
	}
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			id = []byte(strconv.FormatInt(time.Now().UnixNano(), 10))
		}
		value := hex.EncodeToString(id)
		r.Header.Set("X-Request-ID", value)
		w.Header().Set("X-Request-ID", value)
		next.ServeHTTP(w, r)
	})
}

func cors(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
