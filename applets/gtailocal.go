package applets

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/karasz/glibtai"
	//"github.com/karasz/glibtai"
)

func processline(s string) string {
	working := s
	atpos := strings.Index(working, "@")
	if atpos != -1 {
		if len(s) >= atpos+25 {
			lbl := working[atpos:25]
			tn, err := glibtai.TAINfromString(lbl)
			if err == nil {
				working = strings.Replace(working, lbl, fmt.Sprint(glibtai.TAINTime(tn)), 1)
			} else {
				lbl := working[atpos:17] //try TAI
				t, err := glibtai.TAIfromString(lbl)
				if err == nil {
					working = strings.Replace(working, lbl, fmt.Sprint(glibtai.TAITime(t)), 1)
				}
			}
		} else if len(s) >= atpos+17 {
			lbl := working[atpos:17]
			t, err := glibtai.TAIfromString(lbl)
			if err == nil {
				working = strings.Replace(working, lbl, fmt.Sprint(glibtai.TAITime(t)), 1)
			}
		}
	}
	return working
}

func GTAILocalRun(file *os.File) int {

	info, err := file.Stat()

	if err != nil {
		fmt.Println(err)
		return 111
	}
	infomode := info.Mode()
	if infomode&os.ModeNamedPipe == 0 {
		fmt.Println("The command is intended to work with pipes.")
		fmt.Println("Usage: cat logfile | gtailocal")
		return 111
	}
	in := bufio.NewReader(file)
	output := bufio.NewWriter(os.Stdout)
	for {
		input, errr := in.ReadString('\n')
		if errr != nil && errr != io.EOF {
			fmt.Println(errr)
			return 111
		}
		if errr == io.EOF {
			break
		}
		_, errw := output.WriteString(processline(input))
		output.Flush()
		if errw != nil {
			fmt.Println(errw)
			output.Flush()
			return 111
		}

	}
	return 0

}
