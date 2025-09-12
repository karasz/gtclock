package cmd

import (
	"flag"
	"fmt"
	"net"

	"github.com/karasz/glibtai"
	"github.com/karasz/gtclock/gtudpd"
)

const defaultPort = ":4014"

var configDir string

// TAICLOCK Protocol Specification:
//
// The TAICLOCK protocol provides TAI (International Atomic Time) timestamps over UDP.
// It follows DJB's simple time protocol design with TAI timestamps instead of UTC.
//
// Protocol Details:
//   - Transport: UDP
//   - Default Port: 4014
//   - Message Format: Binary, fixed-length
//   - Timestamp Format: TAI64N (12 bytes)
//
// Request Format (20 bytes minimum):
//   Bytes 0-3:  Magic bytes "ctai" (0x63 0x74 0x61 0x69)
//   Bytes 4-19: Client data (typically zeros, ignored by server)
//
// Example Request (hex):
//   63 74 61 69 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
//   |  c  t  a  i |           16 bytes of client data            |
//
// Response Format (20 bytes):
//   Byte  0:     Response marker "s" (0x73)
//   Bytes 1-3:   Unused (copied from request)
//   Bytes 4-11:  TAI64 timestamp (8 bytes, big-endian seconds since TAI epoch)
//   Bytes 12-15: TAI64N nanoseconds (4 bytes, big-endian nanoseconds)
//   Bytes 16-19: Unused (copied from request)
//
// Example Response (hex):
//   73 74 61 69 40 00 00 01 89 AB CD EF 12 34 56 78 00 00 00 00
//   |s t  a  i |     TAI64 timestamp      | nanosecs |  unused  |
//
// TAI vs UTC:
//   - TAI is atomic time without leap seconds
//   - TAI64 epoch: 1970-01-01 00:00:10 TAI (10 seconds after Unix epoch)
//   - Current offset: TAI = UTC + 37 seconds (as of 2025)

var responseHeader = []byte("s")

// sendResponse handles TAIN protocol response
func sendResponse(conn *net.UDPConn, _ int, remoteaddr *net.UDPAddr, buf []byte) {

	copy(buf[0:1], responseHeader)
	taiTime := glibtai.TAINNow()
	copy(buf[4:16], glibtai.TAINPack(taiTime))
	// Send response - ignore errors for performance (UDP is best-effort anyway)
	_, _ = conn.WriteToUDP(buf, remoteaddr)
}

// validateTAINRequest validates TAIN protocol requests
func validateTAINRequest(config *gtudpd.Config) gtudpd.RequestValidator {
	return func(n int, buf []byte, remoteIP net.IP) bool {
		if n < 20 || n > config.MaxRequestSize {
			return false
		}
		if buf[0] != 'c' || buf[1] != 't' || buf[2] != 'a' || buf[3] != 'i' {
			return false
		}

		// Check client permissions (this may involve filesystem operations)
		return config.ClientOK(remoteIP)

	}
}

// GTClockDRun starts a TAIN time server listening on port 4014.
func GTClockDRun(args []string) int {
	fs := flag.NewFlagSet("gtclockd", flag.ContinueOnError)
	fs.StringVar(&configDir, "d", "", "config directory path")

	if err := fs.Parse(args); err != nil {
		_, _ = fmt.Println(err)
		return 111
	}

	config := &gtudpd.Config{
		DefaultPort: defaultPort,
		ConfigDir:   configDir,
	}

	server, err := gtudpd.NewServer(config, sendResponse, validateTAINRequest(config))
	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}
	defer func() { _ = server.Stop() }()

	_, _ = fmt.Printf("TAIN time server listening on %s\n", server.Addr().String())
	server.Start()
	return 0
}
