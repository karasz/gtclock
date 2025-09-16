//go:build darwin

package cmd

import (
	"syscall"
	"time"
)

// setSystemClockTime sets the system clock to the specified time on Darwin (macOS).
// Uses settimeofday with microsecond precision - the standard approach on Darwin.
func setSystemClockTime(t time.Time) error {
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv)
}

// setSystemClock sets the system clock with the given offset on Darwin.
func setSystemClock(offset time.Duration) error {
	t := time.Now().Add(offset)
	return setSystemClockTime(t)
}
