package cmd

import (
	"bytes"
	"fmt"
	"net"

	"github.com/karasz/glibtai"
)

const port = ":4014"

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
	servAddr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		return nil, err
	}

	servConn, err := net.ListenUDP("udp", servAddr)
	if err != nil {
		return nil, err
	}

	return servConn, nil
}

// handleClientRequest processes incoming client requests.
func handleClientRequest(servConn *net.UDPConn, buf []byte) {
	for {
		n, remoteaddr, err := servConn.ReadFromUDP(buf)
		if err != nil {
			_, _ = fmt.Printf("Error  %v", err)
			continue
		}
		if (n >= 20) && (bytes.Equal(buf[:4], []byte("ctai"))) {
			go sendResponse(servConn, remoteaddr, buf)
		}
	}
}

// GTClockDRun starts a TAIN time server listening on port 4014.
func GTClockDRun(_ []string) int {
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
