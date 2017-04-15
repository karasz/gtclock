package main

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/karasz/gtclock/tai64"
)

const port = ":4014"

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func sendResponse(conn *net.UDPConn, addr *net.UDPAddr, b []byte) {
	s := []byte("s")
	copy(b[0:], s)
	copy(b[4:], tai64.TainPack(tai64.TainNow()))
	fmt.Println(tai64.TainUnpack(tai64.TainPack(tai64.TainNow())))
	_, err := conn.WriteToUDP(b, addr)
	if err != nil {
		fmt.Printf("Couldn't send response %v", err)
	}
}

func main() {
	ServAddr, err := net.ResolveUDPAddr("udp", port)
	checkError(err)
	ServConn, err := net.ListenUDP("udp", ServAddr)
	checkError(err)
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
}
