package applets

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
		fmt.Printf("Couldn't send response %v", err)
	}
}

func GTClockDRun(args []string) int {
	ServAddr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		fmt.Println(err)
		return 111
	}
	ServConn, err := net.ListenUDP("udp", ServAddr)
	if err != nil {
		fmt.Println(err)
		return 111
	}
	defer ServConn.Close()

	buf := make([]byte, 256)

	for {
		n, remoteaddr, err := ServConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error  %v", err)
			continue
		} else {
			if (n >= 20) && (bytes.Equal(buf[:4], []byte("ctai"))) {
				go sendResponse(ServConn, remoteaddr, buf)
			}
		}
	}
	return 0
}
