package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateWindow struct {
	start time.Time
	count int
}

type rateLimiter struct {
	mutex   sync.Mutex
	clients map[string]rateWindow
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		clients: make(map[string]rateWindow),
		limit:   limit,
		window:  window,
	}
}

func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		now := time.Now()

		l.mutex.Lock()
		entry := l.clients[host]
		if entry.start.IsZero() || now.Sub(entry.start) >= l.window {
			entry = rateWindow{start: now}
		}
		if entry.count >= l.limit {
			l.mutex.Unlock()
			writeJSON(w, http.StatusTooManyRequests, errorEnvelope{Error: apiError{
				Code:      "rate_limit_exceeded",
				Message:   "At most 10 diagnostic runs may be created per minute.",
				RequestID: r.Header.Get("X-Request-ID"),
			}})
			return
		}
		entry.count++
		l.clients[host] = entry
		l.mutex.Unlock()

		next.ServeHTTP(w, r)
	})
}
