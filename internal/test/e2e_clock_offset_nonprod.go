//go:build !production

package test

import (
	"sync/atomic"
	"time"
)

var e2eClockOffsetSeconds int64

func SetE2EClockOffset(seconds int64) {
	atomic.StoreInt64(&e2eClockOffsetSeconds, seconds)
}

func AdvanceE2EClock(seconds int64) {
	atomic.AddInt64(&e2eClockOffsetSeconds, seconds)
}

func ResetE2EClockOffset() {
	SetE2EClockOffset(0)
}

func E2EClockOffset() time.Duration {
	return time.Duration(atomic.LoadInt64(&e2eClockOffsetSeconds)) * time.Second
}
