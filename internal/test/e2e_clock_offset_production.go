//go:build production

package test

import "time"

func SetE2EClockOffset(_ int64) {}

func AdvanceE2EClock(_ int64) {}

func ResetE2EClockOffset() {}

func E2EClockOffset() time.Duration {
	return 0
}
