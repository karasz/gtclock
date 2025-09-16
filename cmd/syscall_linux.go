//go:build linux

package cmd

import (
	"syscall"
	"time"
)

// setSystemClockTime sets the system clock to the specified time on Linux.
// Uses settimeofday with microsecond precision - reliable across Linux versions.
func setSystemClockTime(t time.Time) error {
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv)
}

// setSystemClock sets the system clock with the given offset on Linux.
func setSystemClock(offset time.Duration) error {
	t := time.Now().Add(offset)
	return setSystemClockTime(t)
}
