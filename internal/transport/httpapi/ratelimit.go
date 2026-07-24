package httpapi

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRateLimitClients = 10_000

type rateWindow struct {
	start time.Time
	count int
}

type rateLimiter struct {
	mutex        sync.Mutex
	clients      map[string]rateWindow
	limit        int
	window       time.Duration
	trustProxy   bool
	requestCount uint64
	now          func() time.Time
}

func newRateLimiter(limit int, window time.Duration, trustProxy bool) *rateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		clients:    make(map[string]rateWindow),
		limit:      limit,
		window:     window,
		trustProxy: trustProxy,
		now:        time.Now,
	}
}

func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := l.now()
		key := clientIP(r, l.trustProxy)

		l.mutex.Lock()
		l.requestCount++
		if l.requestCount%256 == 0 || len(l.clients) >= maxRateLimitClients {
			l.prune(now)
		}
		entry := l.clients[key]
		if entry.start.IsZero() || now.Sub(entry.start) >= l.window {
			entry = rateWindow{start: now}
		}
		resetAfter := time.Until(entry.start.Add(l.window))
		if l.now != nil {
			resetAfter = entry.start.Add(l.window).Sub(now)
		}
		if resetAfter < time.Second {
			resetAfter = time.Second
		}
		remaining := l.limit - entry.count
		if remaining < 0 {
			remaining = 0
		}
		setRateLimitHeaders(w, l.limit, remaining, resetAfter)
		if entry.count >= l.limit {
			l.mutex.Unlock()
			w.Header().Set(
				"Retry-After",
				strconv.Itoa(int(math.Ceil(resetAfter.Seconds()))),
			)
			writeJSON(w, http.StatusTooManyRequests, errorEnvelope{Error: apiError{
				Code:      "rate_limit_exceeded",
				Message:   "Too many requests. Try again shortly.",
				RequestID: r.Header.Get("X-Request-ID"),
			}})
			return
		}
		entry.count++
		l.clients[key] = entry
		setRateLimitHeaders(w, l.limit, l.limit-entry.count, resetAfter)
		l.mutex.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (l *rateLimiter) prune(now time.Time) {
	for key, entry := range l.clients {
		if now.Sub(entry.start) >= l.window {
			delete(l.clients, key)
		}
	}
	if len(l.clients) < maxRateLimitClients {
		return
	}
	var oldestKey string
	var oldest time.Time
	for key, entry := range l.clients {
		if oldest.IsZero() || entry.start.Before(oldest) {
			oldestKey = key
			oldest = entry.start
		}
	}
	delete(l.clients, oldestKey)
}

func setRateLimitHeaders(
	w http.ResponseWriter,
	limit int,
	remaining int,
	resetAfter time.Duration,
) {
	w.Header().Set("RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))
	w.Header().Set(
		"RateLimit-Reset",
		strconv.Itoa(int(math.Ceil(resetAfter.Seconds()))),
	)
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		for _, value := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
			candidate := strings.TrimSpace(value)
			if ip := net.ParseIP(candidate); ip != nil {
				return ip.String()
			}
		}
		if candidate := strings.TrimSpace(r.Header.Get("X-Real-IP")); candidate != "" {
			if ip := net.ParseIP(candidate); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return "unknown"
}
