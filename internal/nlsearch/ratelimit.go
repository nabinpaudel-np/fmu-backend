package nlsearch

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/response"
)

// Every request to this endpoint costs a model call, so the number of calls a
// single client can make is capped. The limit is per signed-in user where
// there is one and per address otherwise, so one user cannot spend another
// user's allowance and a shared address still gets a usable budget.

// RateLimiter is a token bucket per caller.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket

	// refill is the number of tokens added per second.
	refill float64
	// burst is the largest number of tokens a caller can hold.
	burst float64
	// ttl is how long an idle caller is remembered.
	ttl time.Duration

	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter builds a limiter that allows perMinute requests per caller,
// tolerating a burst of burst requests.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	if perMinute < 1 {
		perMinute = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		refill:  float64(perMinute) / 60.0,
		burst:   float64(burst),
		ttl:     10 * time.Minute,
	}
}

// Allow reports whether the caller may make another request.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}

	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.refill)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep drops callers that have not come back for a while, so the map does
// not grow without bound.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for key, b := range l.buckets {
		if now.Sub(b.last) > l.ttl {
			delete(l.buckets, key)
		}
	}
}

// Limit wraps a handler with the rate limit.
func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(callerKey(r)) {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.burst/l.refill)))
			response.Error(w, http.StatusTooManyRequests, "too many search requests, please slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// callerKey identifies the client. A signed-in user is identified by their
// id so that one person's searches cannot exhaust another's allowance.
func callerKey(r *http.Request) string {
	if userID, ok := auth.OptionalUserID(r.Context()); ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + clientIP(r)
}

// clientIP returns the caller's address. chi's RealIP middleware has already
// rewritten RemoteAddr from the forwarding headers by the time this runs, so
// behind a proxy this is the forwarded address. The limit is a ceiling on
// model spend, not a security boundary, which is why a spoofed header only
// costs money rather than access.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
