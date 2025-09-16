//go:build freebsd || openbsd || netbsd

package cmd

import (
	"syscall"
	"time"
)

// setSystemClockTime sets the system clock to the specified time on BSD systems.
// Uses settimeofday with microsecond precision - standard approach on BSD.
func setSystemClockTime(t time.Time) error {
	tv := syscall.NsecToTimeval(t.UnixNano())
	return syscall.Settimeofday(&tv)
}

// setSystemClock sets the system clock with the given offset on BSD systems.
func setSystemClock(offset time.Duration) error {
	t := time.Now().Add(offset)
	return setSystemClockTime(t)
}
