package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
}

type visitor struct {
	lastSeen time.Time
	requests int
}

var rateLimiter = &RateLimiter{
	visitors: make(map[string]*visitor),
}

const (
	maxRequests = 10              // Maximum requests per window
	timeWindow  = 1 * time.Minute // Time window for rate limiting
)

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if !rateLimiter.allowRequest(ip) {
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allowRequest checks if a request from the given IP is allowed
func (rl *RateLimiter) allowRequest(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	rl.cleanup(now)

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			lastSeen: now,
			requests: 1,
		}
		return true
	}

	if now.Sub(v.lastSeen) > timeWindow {
		v.requests = 1
		v.lastSeen = now
		return true
	}

	if v.requests >= maxRequests {
		return false
	}

	v.requests++
	v.lastSeen = now
	return true
}

// cleanup removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanup(now time.Time) {
	for ip, v := range rl.visitors {
		if now.Sub(v.lastSeen) > timeWindow*2 {
			delete(rl.visitors, ip)
		}
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		ww := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		fmt.Printf("%s %s %s %d %v\n",
			r.RemoteAddr, r.Method, r.URL.Path, ww.statusCode, duration)
	})
}

// wrappedWriter wraps http.ResponseWriter to capture status code
type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (ww *wrappedWriter) WriteHeader(statusCode int) {
	ww.statusCode = statusCode
	ww.ResponseWriter.WriteHeader(statusCode)
}

func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	return ip
}
