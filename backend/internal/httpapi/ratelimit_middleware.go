package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipRateLimiter struct {
	mu       sync.Mutex
	limits   map[string][]time.Time
	limit    int
	window   time.Duration
	stopChan chan struct{}
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	limiter := &ipRateLimiter{
		limits:   make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		stopChan: make(chan struct{}),
	}

	// Prune old entries periodically
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for {
			select {
			case <-limiter.stopChan:
				return
			case <-ticker.C:
				limiter.mu.Lock()
				cutoff := time.Now().Add(-window)
				for ip, timestamps := range limiter.limits {
					var valid []time.Time
					for _, ts := range timestamps {
						if ts.After(cutoff) {
							valid = append(valid, ts)
						}
					}
					if len(valid) == 0 {
						delete(limiter.limits, ip)
					} else {
						limiter.limits[ip] = valid
					}
				}
				limiter.mu.Unlock()
			}
		}
	}()

	return limiter
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	var valid []time.Time
	for _, ts := range l.limits[ip] {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= l.limit {
		l.limits[ip] = valid
		return false
	}

	valid = append(valid, now)
	l.limits[ip] = valid
	return true
}

// RateLimitLogin enforces 10 requests per minute per IP for the login endpoint.
func RateLimitLogin() func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(10, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !limiter.allow(ip) {
				w.Header().Set("Retry-After", "60")
				ErrorJSON(w, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan login. Silakan tunggu 1 menit.", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
