package ratelimit

import (
    "encoding/json"
    "net/http"
    "strconv"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type Middleware struct {
    cfg      Config
    limiters sync.Map // key: "<route_key>:<user_id>" → *rate.Limiter
}

func New(cfg Config) *Middleware { return &Middleware{cfg: cfg} }

// Limit returns an http.Handler wrapper that enforces the quota for the
// given route. userExtractor pulls the subject id out of request ctx (the
// IAM middleware sets it before this middleware runs).
func (m *Middleware) Limit(key RouteKey, userExtractor func(*http.Request) string, next http.Handler) http.Handler {
    quota, ok := m.cfg.Quotas[key]
    if !ok {
        return next // no quota configured
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := userExtractor(r)
        if user == "" {
            // No user id → bypass; IAM middleware should have rejected already.
            next.ServeHTTP(w, r)
            return
        }
        lk := string(key) + ":" + user
        lim, _ := m.limiters.LoadOrStore(lk, rate.NewLimiter(rate.Every(time.Minute/time.Duration(quota)), quota))
        l := lim.(*rate.Limiter)
        reservation := l.Reserve()
        if !reservation.OK() {
            writeRateLimitError(w, quota, 60)
            return
        }
        if d := reservation.Delay(); d > 0 {
            reservation.Cancel()
            writeRateLimitError(w, quota, int(d.Seconds())+1)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func writeRateLimitError(w http.ResponseWriter, quota, retryAfterSec int) {
    w.Header().Set("content-type", "application/json")
    w.Header().Set("retry-after", strconv.Itoa(retryAfterSec))
    w.WriteHeader(http.StatusTooManyRequests)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "error":               "rate_limited",
        "quota_per_minute":    quota,
        "retry_after_seconds": retryAfterSec,
    })
}
