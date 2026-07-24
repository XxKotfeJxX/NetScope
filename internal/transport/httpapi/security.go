package httpapi

import (
	"net/http"
	"strings"
)

func securityHeaders(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set(
				"Permissions-Policy",
				"camera=(), geolocation=(), microphone=()",
			)
			w.Header().Set(
				"Content-Security-Policy",
				"default-src 'none'; base-uri 'none'; frame-ancestors 'none'",
			)
			w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
			if production {
				w.Header().Set(
					"Strict-Transport-Security",
					"max-age=31536000; includeSubDomains",
				)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func csrfOriginGuard(enabled bool, origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled || isSafeMethod(r.Method) || !usesSessionCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			if strings.TrimSpace(r.Header.Get("Origin")) != origin {
				writeJSON(w, http.StatusForbidden, errorEnvelope{Error: apiError{
					Code:      "origin_rejected",
					Message:   "The request origin is not allowed.",
					RequestID: r.Header.Get("X-Request-ID"),
				}})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func usesSessionCookie(r *http.Request) bool {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return false
	}
	cookie, err := r.Cookie(sessionCookieName)
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}

func cors(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestOrigin := strings.TrimSpace(r.Header.Get("Origin"))
			if requestOrigin == "" || requestOrigin == origin {
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
			}
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				if requestOrigin != "" && requestOrigin != origin {
					writeJSON(w, http.StatusForbidden, errorEnvelope{Error: apiError{
						Code:      "origin_rejected",
						Message:   "The request origin is not allowed.",
						RequestID: r.Header.Get("X-Request-ID"),
					}})
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
