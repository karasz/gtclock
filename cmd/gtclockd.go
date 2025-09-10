package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/karasz/glibtai"
)

const defaultPort = ":4014"

var configDir string

// getPort reads the port configuration from the config directory.
// Returns the port in ":port" format, falling back to defaultPort if not configured.
func getPort() string {
	if configDir == "" {
		return defaultPort
	}

	portFile := filepath.Join(configDir, "port")
	data, err := os.ReadFile(portFile)
	if err != nil {
		return defaultPort
	}

	portStr := strings.TrimSpace(string(data))
	if portStr == "" {
		return defaultPort
	}

	// Handle port with colon prefix
	var portNum string
	if portStr[0] == ':' {
		portNum = portStr[1:]
	} else {
		portNum = portStr
	}

	// Validate port number
	if port, err := strconv.Atoi(portNum); err != nil || port <= 0 || port > 65535 {
		return defaultPort
	}

	if portStr[0] != ':' {
		return ":" + portStr
	}
	return portStr
}

// checkNetworkFile checks if a network file exists in the config directory.
func checkNetworkFile(network string) bool {
	networkFile := filepath.Join(configDir, network)
	_, err := os.Stat(networkFile)
	return err == nil
}

// checkIPv4Networks checks IPv4 network patterns for clientok.
func checkIPv4Networks(ip4 net.IP) bool {
	// Check /8 network
	network8 := strconv.Itoa(int(ip4[0]))
	if checkNetworkFile(network8) {
		return true
	}

	// Check /16 network
	network16 := strconv.Itoa(int(ip4[0])) + "." + strconv.Itoa(int(ip4[1]))
	if checkNetworkFile(network16) {
		return true
	}

	// Check /24 network
	network24 := strconv.Itoa(int(ip4[0])) + "." + strconv.Itoa(int(ip4[1])) + "." + strconv.Itoa(int(ip4[2]))
	return checkNetworkFile(network24)
}

// clientok checks if the given IP address is allowed to use the server.
// It follows DJB's clientok pattern by checking for the existence of files
// in the config directory matching the IP address.
func clientok(ip net.IP) bool {
	if configDir == "" {
		return true // If no config directory specified, allow all
	}

	// For IPv4, check for network matches starting with broadest (/8)
	if ip4 := ip.To4(); ip4 != nil {
		if checkIPv4Networks(ip4) {
			return true
		}
	}

	// Check for exact IP match last
	return checkNetworkFile(ip.String())
}

func sendResponse(conn *net.UDPConn, addr *net.UDPAddr, b []byte) {
	s := []byte("s")
	copy(b[0:], s)
	copy(b[4:], glibtai.TAINPack(glibtai.TAINNow()))
	_, err := conn.WriteToUDP(b, addr)
	if err != nil {
		_, _ = fmt.Printf("Couldn't send response %v", err)
	}
}

// setupUDPServer creates and returns a UDP server connection.
func setupUDPServer() (*net.UDPConn, error) {
	port := getPort()
	servAddr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		return nil, err
	}

	servConn, err := net.ListenUDP("udp", servAddr)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			if configDir == "" {
				return nil, fmt.Errorf(
					"permission denied binding to port %s - try running as root or use a port >= 1024 "+
						"(configure with -d flag and create port file)", port)
			}
			return nil, fmt.Errorf(
				"permission denied binding to port %s - try running as root or use a port >= 1024 "+
					"(configure in %s/port)", port, configDir)
		}
		return nil, err
	}

	return servConn, nil
}

// isValidRequest checks if the request is valid TAIN request.
func isValidRequest(n int, buf []byte) bool {
	return (n >= 20) && (bytes.Equal(buf[:4], []byte("ctai")))
}

// handleClientRequest processes incoming client requests.
func handleClientRequest(servConn *net.UDPConn, buf []byte) {
	for {
		n, remoteaddr, err := servConn.ReadFromUDP(buf)
		if err != nil {
			_, _ = fmt.Printf("Error  %v", err)
			continue
		}
		if isValidRequest(n, buf) && clientok(remoteaddr.IP) {
			go sendResponse(servConn, remoteaddr, buf)
		}
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

	servConn, err := setupUDPServer()
	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}
	defer func() { _ = servConn.Close() }()

	buf := make([]byte, 256)
	handleClientRequest(servConn, buf)
	return 0
}
