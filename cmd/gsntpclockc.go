// Package cmd provides the implementation for various gtclock functionalities.
package cmd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

type mode byte
type ntpTime uint64

const (
	reserved mode = 0 + iota
	symmetricActive
	symmetricPassive
	client
	server
	broadcast
	controlMessage
	reservedPrivate
)
const nanoPerSec = 1e9

var ntpEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

// Duration interprets the fixed-point ntpTime as a number of elapsed seconds
// and returns the corresponding time.Duration value.
func (t ntpTime) duration() time.Duration {
	sec := (t >> 32) * nanoPerSec
	frac := (t & 0xffffffff) * nanoPerSec >> 32
	return time.Duration(sec + frac)
}

// Decode interprets the fixed-point ntpTime and returns a time.Time
func (t ntpTime) decode() time.Time {
	return ntpEpoch.Add(t.duration())
}

// Encode encodes a time.Time in a ntpTime format
func encode(t time.Time) ntpTime {
	nsec := uint64(t.Sub(ntpEpoch))
	sec := nsec / nanoPerSec
	frac := (nsec - sec*nanoPerSec) << 32 / nanoPerSec
	return ntpTime(sec<<32 | frac)
}

// Sub subtracts two ntpTime values
func (t ntpTime) sub(tt ntpTime) time.Duration {
	return t.decode().Sub(tt.decode())
}

type msg struct {
	LiVnMode       byte // Leap Indicator (2) + Version (3) + Mode (3)
	Stratum        byte
	Poll           byte
	Precision      byte
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	ReferenceTime  ntpTime
	OriginateTime  ntpTime
	ReceiveTime    ntpTime
	TransmitTime   ntpTime
}

// SetVersion sets the NTP protocol version on the message.
func (m *msg) setVersion(v byte) {
	m.LiVnMode = (m.LiVnMode & 0xc7) | v<<3
}

// SetMode sets the NTP protocol mode on the message.
func (m *msg) setMode(md mode) {
	m.LiVnMode = (m.LiVnMode & 0xf8) | byte(md)
}

// GetTime returns the "receive time" from the remote NTP server
// specified as host.  NTP client mode is used.
func getTime(host string) (msg, ntpTime, error) {
	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, "123"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return msg{}, 0, err
	}

	con, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return msg{}, 0, err
	}

	defer func() { _ = con.Close() }()
	_ = con.SetDeadline(time.Now().Add(5 * time.Second))

	m := new(msg)
	m.setMode(client)
	m.setVersion(4)
	m.TransmitTime = encode(time.Now())

	err = binary.Write(con, binary.BigEndian, m)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return msg{}, 0, err
	}

	err = binary.Read(con, binary.BigEndian, m)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return msg{}, 0, err
	}

	dest := encode(time.Now())

	return *m, dest, nil
}

// getParams returns two time.Durations the time offset and rtt time
func getParams(m msg, dest ntpTime) (offset time.Duration, rtt time.Duration) {
	t1 := m.OriginateTime
	t2 := m.ReceiveTime
	t3 := m.TransmitTime
	t4 := dest
	offset = (t2.sub(t1) + t3.sub(t4)) / 2
	rtt = t4.sub(t1) - t2.sub(t3)
	return offset, rtt
}

// parseNTPArgs parses command line arguments for NTP client.
func parseNTPArgs(args []string) (servIP net.IP, saveClock bool, err error) {
	if len(args) == 0 {
		return nil, false, errors.New("usage: gntpclock <server_ip> [saveclock]")
	}

	switch len(args) {
	case 1:
		servIP = net.ParseIP(args[0])
		if servIP == nil {
			return nil, false, fmt.Errorf("invalid IP address: %s", args[0])
		}
	case 2:
		servIP = net.ParseIP(args[0])
		if servIP == nil {
			return nil, false, fmt.Errorf("invalid IP address: %s", args[0])
		}
		saveClock = args[1] == "saveclock"
	default:
		return nil, false, errors.New("usage: gntpclock <server_ip> [saveclock]")
	}
	return servIP, saveClock, nil
}

// GSNTPClockCRun implements SNTP client functionality for time synchronization.
func GSNTPClockCRun(args []string) int {
	servIP, saveClock, err := parseNTPArgs(args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return 111
	}

	m, dst, err := getTime(servIP.String())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return 111
	}

	offset, _ := getParams(m, dst)

	if saveClock {
		if err := setSystemClock(offset); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			return 111
		}
	}

	return 0
}
