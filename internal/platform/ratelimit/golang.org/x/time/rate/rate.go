package rate

import (
	"math"
	"sync"
	"time"
)

type Limit float64

const Inf = Limit(math.MaxFloat64)

func Every(interval time.Duration) Limit {
	if interval <= 0 {
		return Inf
	}
	return 1 / Limit(interval.Seconds())
}

type Limiter struct {
	mu     sync.Mutex
	limit  Limit
	burst  float64
	tokens float64
	last   time.Time
}

func NewLimiter(r Limit, burst int) *Limiter {
	if burst < 0 {
		burst = 0
	}
	return &Limiter{
		limit:  r,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (l *Limiter) Reserve() *Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.advance(now)

	if l.burst < 1 {
		return &Reservation{ok: false}
	}
	if l.tokens >= 1 {
		l.tokens -= 1
		return &Reservation{ok: true}
	}
	if l.limit <= 0 {
		return &Reservation{ok: false}
	}

	need := 1 - l.tokens
	delaySec := need / float64(l.limit)
	if delaySec < 0 {
		delaySec = 0
	}
	delay := time.Duration(delaySec * float64(time.Second))
	l.tokens -= 1

	return &Reservation{
		ok:     true,
		delay:  delay,
		lim:    l,
		tokens: 1,
	}
}

func (l *Limiter) advance(now time.Time) {
	if now.Before(l.last) {
		return
	}
	if l.limit <= 0 {
		l.last = now
		return
	}
	elapsed := now.Sub(l.last).Seconds()
	l.tokens += elapsed * float64(l.limit)
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	l.last = now
}

type Reservation struct {
	ok     bool
	delay  time.Duration
	lim    *Limiter
	tokens float64
}

func (r *Reservation) OK() bool { return r != nil && r.ok }

func (r *Reservation) Delay() time.Duration {
	if r == nil || !r.ok {
		return 0
	}
	if r.delay < 0 {
		return 0
	}
	return r.delay
}

func (r *Reservation) Cancel() {
	if r == nil || !r.ok || r.lim == nil || r.tokens == 0 {
		return
	}

	r.lim.mu.Lock()
	defer r.lim.mu.Unlock()

	r.lim.advance(time.Now())
	r.lim.tokens += r.tokens
	if r.lim.tokens > r.lim.burst {
		r.lim.tokens = r.lim.burst
	}
	r.tokens = 0
}
