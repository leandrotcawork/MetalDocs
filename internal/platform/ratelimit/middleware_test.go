package ratelimit_test

import (
    "net/http"
    "net/http/httptest"
    "strconv"
    "testing"

    "metaldocs/internal/platform/ratelimit"
)

func TestLimit_BurstThenRejects(t *testing.T) {
    mw := ratelimit.New(ratelimit.Config{Quotas: map[ratelimit.RouteKey]int{ratelimit.RouteExportPDF: 3}})
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
    h := mw.Limit(ratelimit.RouteExportPDF, func(r *http.Request) string { return "u1" }, next)

    // 3 permitted, 4th rejected.
    for i := 0; i < 3; i++ {
        rr := httptest.NewRecorder()
        h.ServeHTTP(rr, httptest.NewRequest("POST", "/x", nil))
        if rr.Code != 204 {
            t.Fatalf("req %d: want 204, got %d", i, rr.Code)
        }
    }
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, httptest.NewRequest("POST", "/x", nil))
    if rr.Code != http.StatusTooManyRequests {
        t.Fatalf("4th req: want 429, got %d", rr.Code)
    }
    retry := rr.Header().Get("retry-after")
    if n, _ := strconv.Atoi(retry); n < 1 {
        t.Fatalf("retry-after must be ≥1s, got %q", retry)
    }
}

func TestLimit_PerUserIsolation(t *testing.T) {
    mw := ratelimit.New(ratelimit.Config{Quotas: map[ratelimit.RouteKey]int{ratelimit.RouteExportPDF: 1}})
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
    h := mw.Limit(ratelimit.RouteExportPDF, func(r *http.Request) string { return r.Header.Get("x-user") }, next)

    // User A burns its 1 token.
    rrA := httptest.NewRecorder()
    reqA := httptest.NewRequest("POST", "/x", nil); reqA.Header.Set("x-user", "A")
    h.ServeHTTP(rrA, reqA)
    if rrA.Code != 204 { t.Fatalf("A first: want 204, got %d", rrA.Code) }

    rrA2 := httptest.NewRecorder()
    reqA2 := httptest.NewRequest("POST", "/x", nil); reqA2.Header.Set("x-user", "A")
    h.ServeHTTP(rrA2, reqA2)
    if rrA2.Code != http.StatusTooManyRequests { t.Fatalf("A second: want 429, got %d", rrA2.Code) }

    // User B still has its own bucket.
    rrB := httptest.NewRecorder()
    reqB := httptest.NewRequest("POST", "/x", nil); reqB.Header.Set("x-user", "B")
    h.ServeHTTP(rrB, reqB)
    if rrB.Code != 204 { t.Fatalf("B first: want 204, got %d", rrB.Code) }
}
