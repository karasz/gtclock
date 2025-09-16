//go:build windows

package cmd

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetSystemTimeAdjustment = kernel32.NewProc("SetSystemTimeAdjustment")
	procGetSystemTimeAdjustment = kernel32.NewProc("GetSystemTimeAdjustment")
	procSetSystemTime           = kernel32.NewProc("SetSystemTime")
)

// Windows uses 100-nanosecond intervals for high-precision time operations

// setSystemClockTime sets the system clock to the specified time on Windows.
// Uses high-precision time adjustment when possible, falls back to SetSystemTime.
func setSystemClockTime(t time.Time) error {
	// Try high-precision time adjustment first
	if err := setSystemTimeWithAdjustment(t); err == nil {
		return nil
	}

	// Fall back to standard SetSystemTime (millisecond precision)
	return setSystemTimeStandard(t)
}

// setSystemTimeWithAdjustment attempts to set time using time adjustment APIs
func setSystemTimeWithAdjustment(targetTime time.Time) error {
	var timeAdjustment, timeIncrement uint32
	var timeAdjustmentDisabled uint8

	// Get current time adjustment settings
	r1, _, err := procGetSystemTimeAdjustment.Call(
		uintptr(unsafe.Pointer(&timeAdjustment)),
		uintptr(unsafe.Pointer(&timeIncrement)),
		uintptr(unsafe.Pointer(&timeAdjustmentDisabled)),
	)
	if r1 == 0 {
		return err
	}

	// Calculate the time difference in 100-nanosecond units
	now := time.Now()
	diff := targetTime.Sub(now)
	adjustment := int64(diff.Nanoseconds() / 100) // Convert to 100-nanosecond units

	// If difference is small enough, use time adjustment
	if adjustment > -86400*10000000 && adjustment < 86400*10000000 { // Within 1 day
		newAdjustment := uint32(int64(timeAdjustment) + adjustment)
		r2, _, err2 := procSetSystemTimeAdjustment.Call(
			uintptr(newAdjustment),
			0, // FALSE - enable time adjustment
		)
		if r2 != 0 {
			return nil
		}
		return err2
	}

	return syscall.EINVAL // Difference too large for adjustment
}

// setSystemTimeStandard uses the standard SetSystemTime API (millisecond precision)
func setSystemTimeStandard(t time.Time) error {
	utc := t.UTC()

	// SYSTEMTIME structure
	st := [8]uint16{
		uint16(utc.Year()),
		uint16(utc.Month()),
		uint16(utc.Weekday()),
		uint16(utc.Day()),
		uint16(utc.Hour()),
		uint16(utc.Minute()),
		uint16(utc.Second()),
		uint16(utc.Nanosecond() / 1000000), // Milliseconds
	}

	r1, _, err := procSetSystemTime.Call(uintptr(unsafe.Pointer(&st[0])))
	if r1 == 0 {
		return err
	}
	return nil
}

// setSystemClock sets the system clock with the given offset on Windows.
func setSystemClock(offset time.Duration) error {
	t := time.Now().Add(offset)
	return setSystemClockTime(t)
}
