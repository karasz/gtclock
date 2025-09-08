package cmd

import (
	"fmt"
	"os"
)

// MainDispatcher is called if run as "gtclock <subcommand>"
func MainDispatcher(args []string) int {
	ret := 0
	if len(args) == 0 {
		_, _ = fmt.Println("Available applets: gtailocal,gtclockd,gtclockc,gntpclock")
		ret = 1
		return ret
	}

	switch args[0] {
	case "gtailocal":
		ret = GTAILocalRun(args[1:])
	case "gtclockd":
		ret = GTClockDRun(args[1:])
	case "gtclockc":
		ret = GTClockCRun(args[1:])
	case "gntpclock":
		ret = GNTPClockRun(args[1:])

	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		ret = 1
	}
	return ret
}
