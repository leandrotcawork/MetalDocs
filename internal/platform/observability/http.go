package observability

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	authdomain "metaldocs/internal/modules/auth/domain"
)

type routeMetrics struct {
	requests   uint64
	errors     uint64
	durationMs uint64
}

type HTTPObservability struct {
	logger          *slog.Logger
	runtimeProvider RuntimeStatusProvider
	mu              sync.RWMutex
	byKey           map[string]*routeMetrics
}

type metricItem struct {
	Route           string `json:"route"`
	Method          string `json:"method"`
	Requests        uint64 `json:"requests"`
	Errors          uint64 `json:"errors"`
	DurationTotalMs uint64 `json:"durationTotalMs"`
	AvgDurationMs   uint64 `json:"avgDurationMs"`
}

func NewHTTPObservability(runtimeProvider ...RuntimeStatusProvider) *HTTPObservability {
	var provider RuntimeStatusProvider
	if len(runtimeProvider) > 0 {
		provider = runtimeProvider[0]
	}
	return &HTTPObservability{
		logger:          slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
		runtimeProvider: provider,
		byKey:           make(map[string]*routeMetrics),
	}
}

func (o *HTTPObservability) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		route := normalizeRoute(r.URL.Path)
		method := r.Method
		elapsedMs := time.Since(start).Milliseconds()
		if elapsedMs < 0 {
			elapsedMs = 0
		}
		durationMs := uint64(elapsedMs)
		isError := sw.status >= 400

		m := o.getMetric(route, method)
		atomic.AddUint64(&m.requests, 1)
		if isError {
			atomic.AddUint64(&m.errors, 1)
		}
		atomic.AddUint64(&m.durationMs, durationMs)

		traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
		if traceID == "" {
			traceID = "trace-local"
		}
		userID := "anonymous"
		if currentUser, ok := authdomain.CurrentUserFromContext(r.Context()); ok && strings.TrimSpace(currentUser.UserID) != "" {
			userID = currentUser.UserID
		}

		o.logger.Info("http_request",
			"trace_id", traceID,
			"user_id", userID,
			"method", method,
			"path", r.URL.Path,
			"route", route,
			"status", sw.status,
			"duration_ms", durationMs,
		)
	})
}

func (o *HTTPObservability) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		items := o.snapshot()
		payload := map[string]any{"items": items}
		if o.runtimeProvider != nil {
			payload["runtime"] = o.runtimeProvider.RuntimeMetrics(r.Context())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
}

func (o *HTTPObservability) snapshot() []metricItem {
	o.mu.RLock()
	defer o.mu.RUnlock()

	out := make([]metricItem, 0, len(o.byKey))
	for key, m := range o.byKey {
		method, route := splitKey(key)
		req := atomic.LoadUint64(&m.requests)
		errs := atomic.LoadUint64(&m.errors)
		dur := atomic.LoadUint64(&m.durationMs)
		avg := uint64(0)
		if req > 0 {
			avg = dur / req
		}
		out = append(out, metricItem{
			Route:           route,
			Method:          method,
			Requests:        req,
			Errors:          errs,
			DurationTotalMs: dur,
			AvgDurationMs:   avg,
		})
	}
	return out
}

func (o *HTTPObservability) getMetric(route, method string) *routeMetrics {
	key := method + " " + route
	o.mu.RLock()
	m, ok := o.byKey[key]
	o.mu.RUnlock()
	if ok {
		return m
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if existing, ok := o.byKey[key]; ok {
		return existing
	}
	created := &routeMetrics{}
	o.byKey[key] = created
	return created
}

func normalizeRoute(path string) string {
	if strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/versions") {
		return "/api/v1/documents/{documentId}/versions"
	}
	if strings.HasPrefix(path, "/api/v1/workflow/documents/") && strings.HasSuffix(path, "/transitions") {
		return "/api/v1/workflow/documents/{documentId}/transitions"
	}
	if strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/roles") {
		return "/api/v1/iam/users/{userId}/roles"
	}
	if strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/reset-password") {
		return "/api/v1/iam/users/{userId}/reset-password"
	}
	if strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/unlock") {
		return "/api/v1/iam/users/{userId}/unlock"
	}
	return path
}

func splitKey(key string) (method, route string) {
	parts := strings.SplitN(key, " ", 2)
	if len(parts) != 2 {
		return "UNKNOWN", key
	}
	return parts[0], parts[1]
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		if v := w.Header().Get("Status"); v != "" {
			if code, err := strconv.Atoi(v); err == nil {
				w.status = code
			}
		}
		if w.status == 0 {
			w.status = http.StatusOK
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
