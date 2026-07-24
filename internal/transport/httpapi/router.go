package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/XxKotfeJxX/netscope/internal/reports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Logger              *slog.Logger
	Pool                *pgxpool.Pool
	Version             string
	WebOrigin           string
	Runs                *diagnostics.Service
	Events              diagnostics.EventPublisher
	Runtime             RuntimeInfo
	Checks              map[string]Capability
	Monitoring          *monitoring.Service
	Identity            *identity.Service
	Collaboration       *collaboration.Service
	Reports             *reports.Service
	SessionCookieSecure bool
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
		runtime: deps.Runtime, checks: deps.Checks, monitoring: deps.Monitoring,
	}
	limiter := newRateLimiter(10, time.Minute)
	router.Route("/api/v1", func(router chi.Router) {
		router.Get("/capabilities", api.capabilities)
		if deps.Reports != nil {
			publicReports := reportsHandler{service: deps.Reports}
			router.With(limiter.middleware).Get(
				"/public/reports/{token}",
				publicReports.publicReport,
			)
		}
		if deps.Identity != nil {
			accounts := identityHandler{
				service: deps.Identity, cookieSecure: deps.SessionCookieSecure,
			}
			router.With(limiter.middleware).Post("/auth/register", accounts.register)
			router.With(limiter.middleware).Post("/auth/login", accounts.login)
			router.Post("/auth/logout", accounts.logout)
			router.Group(func(router chi.Router) {
				router.Use(accounts.requireAccount)
				router.Get("/me", accounts.me)
				router.Get("/workspaces", accounts.listWorkspaces)
				router.Post("/workspaces", accounts.createWorkspace)
			})
			router.Group(func(router chi.Router) {
				router.Use(accounts.requireAccount)
				router.Use(accounts.requireWorkspace)
				mountWorkspaceRoutes(
					router,
					api,
					limiter,
					accounts.requireRole,
					deps.Collaboration,
					deps.Reports,
				)
			})
		} else {
			mountWorkspaceRoutes(router, api, limiter, nil, nil, nil)
		}
	})

	return router
}

type roleGuard func(identity.Role) func(http.Handler) http.Handler

func mountWorkspaceRoutes(
	router chi.Router,
	api apiHandler,
	limiter *rateLimiter,
	guard roleGuard,
	collaborationService *collaboration.Service,
	reportsService *reports.Service,
) {
	operator := func(middlewares ...func(http.Handler) http.Handler) []func(
		http.Handler,
	) http.Handler {
		result := make([]func(http.Handler) http.Handler, 0, len(middlewares)+1)
		if guard != nil {
			result = append(result, guard(identity.RoleOperator))
		}
		return append(result, middlewares...)
	}

	router.With(operator(limiter.middleware)...).Post("/runs", api.createRun)
	router.Get("/runs", api.listRuns)
	router.Get("/runs/{id}", api.getRun)
	router.With(operator()...).Post("/runs/{id}/cancel", api.cancelRun)
	router.Get("/runs/{id}/events", api.runEvents)
	router.Get("/runs/{id}/export", api.exportRun)
	router.Get("/targets", api.listTargets)
	router.With(operator(limiter.middleware)...).Post("/targets", api.createTarget)
	router.Get("/targets/{targetID}", api.getTarget)
	router.With(operator()...).Put("/targets/{targetID}", api.updateTarget)
	router.With(operator()...).Delete("/targets/{targetID}", api.deleteTarget)
	router.With(operator()...).Post("/targets/{targetID}/pause", api.pauseTarget)
	router.With(operator()...).Post("/targets/{targetID}/resume", api.resumeTarget)
	router.Get("/targets/{targetID}/checks", api.listTargetChecks)
	router.Get("/targets/{targetID}/maintenance", api.listMaintenanceWindows)
	router.With(operator()...).Post(
		"/targets/{targetID}/maintenance",
		api.createMaintenanceWindow,
	)
	router.With(operator()...).Delete(
		"/targets/{targetID}/maintenance/{windowID}",
		api.deleteMaintenanceWindow,
	)
	router.Get("/targets/{targetID}/notifications", api.listNotificationChannels)
	router.With(operator()...).Post(
		"/targets/{targetID}/notifications",
		api.createNotificationChannel,
	)
	router.With(operator()...).Delete(
		"/targets/{targetID}/notifications/{channelID}",
		api.deleteNotificationChannel,
	)
	router.Get("/monitoring", api.monitoringOverview)
	if collaborationService != nil {
		collaborationAPI := collaborationHandler{service: collaborationService}
		router.With(guard(identity.RoleAdmin)).Get(
			"/workspace/members",
			collaborationAPI.listMembers,
		)
		router.With(guard(identity.RoleAdmin)).Post(
			"/workspace/members",
			collaborationAPI.addMember,
		)
		router.With(guard(identity.RoleAdmin)).Patch(
			"/workspace/members/{userID}",
			collaborationAPI.updateMember,
		)
		router.With(guard(identity.RoleAdmin)).Delete(
			"/workspace/members/{userID}",
			collaborationAPI.removeMember,
		)
		router.With(guard(identity.RoleAdmin)).Get(
			"/workspace/audit",
			collaborationAPI.listAudit,
		)
		router.With(guard(identity.RoleAdmin)).Get(
			"/workspace/api-keys",
			collaborationAPI.listAPIKeys,
		)
		router.With(guard(identity.RoleAdmin)).Post(
			"/workspace/api-keys",
			collaborationAPI.createAPIKey,
		)
		router.With(guard(identity.RoleAdmin)).Delete(
			"/workspace/api-keys/{keyID}",
			collaborationAPI.revokeAPIKey,
		)
	}
	if reportsService != nil {
		reportAPI := reportsHandler{service: reportsService}
		router.Get("/runs/{id}/comments", reportAPI.listComments)
		router.With(operator()...).Post(
			"/runs/{id}/comments",
			reportAPI.createComment,
		)
		router.With(operator()...).Delete(
			"/runs/{id}/comments/{commentID}",
			reportAPI.deleteComment,
		)
		router.Get("/runs/{id}/public-links", reportAPI.listPublicLinks)
		router.With(operator()...).Post(
			"/runs/{id}/public-links",
			reportAPI.createPublicLink,
		)
		router.With(operator()...).Delete(
			"/runs/{id}/public-links/{linkID}",
			reportAPI.revokePublicLink,
		)
	}
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
				"path", safeRequestPath(r.URL.Path),
				"status", recorder.status,
				"duration_ms", time.Since(started).Milliseconds(),
			)
		})
	}
}

func safeRequestPath(path string) string {
	if strings.HasPrefix(path, "/api/v1/public/reports/") {
		return "/api/v1/public/reports/[redacted]"
	}
	return path
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
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Accept, Authorization, Content-Type, X-Request-ID, X-Workspace-ID",
			)
			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
