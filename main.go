package main

import (
	"fmt"
	"os"
	"path/filepath"

	app "github.com/karasz/gtclock/applets"
)

func main() {
	_, calledAs := filepath.Split(os.Args[0])
	args := os.Args[1:]
	res := 0
	switch calledAs {
	case "gtclock":
		res = app.GTClockRun(args)
	case "gtclockd":
		res = app.GTClockDRun(args)
	case "gntpclock":
		res = app.GNTPClockRun(args)
	default:
		fmt.Println("Called as ", calledAs, ". I don't recognize that name")
	}
	os.Exit(res)
}
