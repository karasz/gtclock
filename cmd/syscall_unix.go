//go:build unix && !windows && !linux && !darwin && !freebsd && !openbsd && !netbsd

package cmd

import (
	"syscall"
	"time"
)

// setSystemClockTime sets the system clock to the specified time on Unix systems.
func setSystemClockTime(t time.Time) error {
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv)
}

// setSystemClock sets the system clock with the given offset on Unix systems.
func setSystemClock(offset time.Duration) error {
	t := time.Now().Add(offset)
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv)
}
