// Package main provides the gtclock multi-binary implementation.
// gtclock can run as different programs based on the name it's called with:
// gtclock, gtclockd, gntpclock, or gtailocal.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karasz/gtclock/cmd"
)

func main() {
	app := filepath.Base(os.Args[0])

	commands := map[string]func(args []string) int{
		"gtclock":   cmd.MainDispatcher, // fallback dispatcher (like busybox)
		"gtailocal": cmd.GTAILocalRun,
		"gtclockd":  cmd.GTClockDRun,
		"gtclockc":  cmd.GTClockCRun,
		"gntpclock": cmd.GNTPClockRun,
	}

	// Check if command exists
	fn, ok := commands[app]

	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n", app)
		os.Exit(1)
	}
	fn(os.Args[1:])
}
