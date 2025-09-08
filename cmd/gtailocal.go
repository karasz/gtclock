package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/karasz/glibtai"
)

// tryParseTimestamp attempts to parse and replace a timestamp at the given position.
func tryParseTimestamp(working string, atpos int, length int) (string, bool) {
	if len(working) < atpos+length {
		return working, false
	}

	lbl := working[atpos : atpos+length]

	if length == 25 {
		// Try TAIN format first
		if tn, err := glibtai.TAINfromString(lbl); err == nil {
			return strings.Replace(working, lbl, fmt.Sprint(glibtai.TAINTime(tn)), 1), true
		}
	}

	if length >= 17 {
		// Try TAI format
		if t, err := glibtai.TAIfromString(lbl[:17]); err == nil {
			return strings.Replace(working, lbl[:17], fmt.Sprint(glibtai.TAITime(t)), 1), true
		}
	}

	return working, false
}

func processline(s string) string {
	working := s
	atpos := strings.Index(working, "@")
	if atpos == -1 {
		return working
	}

	// Try TAIN format (25 chars) first
	if result, ok := tryParseTimestamp(working, atpos, 25); ok {
		return result
	}

	// Try TAI format (17 chars)
	if result, ok := tryParseTimestamp(working, atpos, 17); ok {
		return result
	}

	return working
}

// validateInputFile checks if the input file is suitable for processing.
func validateInputFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeNamedPipe == 0 {
		return errors.New("the command is intended to work with pipes.\nUsage: cat logfile | gtailocal")
	}

	return nil
}

// processAndWriteLine processes a single line and writes it to output.
func processAndWriteLine(line string, output *bufio.Writer) error {
	processed := processline(line)
	if _, err := output.WriteString(processed); err != nil {
		return err
	}
	return output.Flush()
}

// processInputStream reads from input and writes processed lines to output.
func processInputStream(in *bufio.Reader, output *bufio.Writer) error {
	for {
		input, err := in.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := processAndWriteLine(input, output); err != nil {
			return err
		}
	}

	return nil
}

// GTAILocalRun converts TAI and TAIN timestamps to RFC3339 format from standard input.
func GTAILocalRun(args []string) int {
	file := os.Stdin
	if len(args) > 0 && args[0] != "-" {
		// In future, could support file input here, but for now we bail out
		_, _ = fmt.Println("we do not support calling filenames yet")
		return 111
	}

	if err := validateInputFile(file); err != nil {
		_, _ = fmt.Println(err)
		return 111
	}

	in := bufio.NewReader(file)
	output := bufio.NewWriter(os.Stdout)

	if err := processInputStream(in, output); err != nil {
		_, _ = fmt.Println(err)
		_ = output.Flush()
		return 111
	}

	return 0
}
