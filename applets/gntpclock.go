package applets

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
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
func (t ntpTime) Duration() time.Duration {
	sec := (t >> 32) * nanoPerSec
	frac := (t & 0xffffffff) * nanoPerSec >> 32
	return time.Duration(sec + frac)
}

// Decode interprets the fixed-point ntpTime and returns a time.Time
func (t ntpTime) Decode() time.Time {
	return ntpEpoch.Add(t.Duration())
}

// Encode encodes a time.Time in a ntpTime format
func Encode(t time.Time) ntpTime {
	nsec := uint64(t.Sub(ntpEpoch))
	sec := nsec / nanoPerSec
	frac := (nsec - sec*nanoPerSec) << 32 / nanoPerSec
	return ntpTime(sec<<32 | frac)
}

// Sub subtracts two ntpTime values
func (t ntpTime) Sub(tt ntpTime) time.Duration {
	return t.Decode().Sub(tt.Decode())
}

type msg struct {
	LiVnMode       byte // Leap Indicator (2) + Version (3) + Mode (3)
	Stratum        byte
	Poll           byte
	Precision      byte
	RootDelay      uint32
	RootDispersion uint32
	ReferenceId    uint32
	ReferenceTime  ntpTime
	OriginateTime  ntpTime
	ReceiveTime    ntpTime
	TransmitTime   ntpTime
}

// SetVersion sets the NTP protocol version on the message.
func (m *msg) SetVersion(v byte) {
	m.LiVnMode = (m.LiVnMode & 0xc7) | v<<3
}

// SetMode sets the NTP protocol mode on the message.
func (m *msg) SetMode(md mode) {
	m.LiVnMode = (m.LiVnMode & 0xf8) | byte(md)
}

// durtoTV transforms a time.Duration in two 64 bit integers
// suitable for setting Timevalue values
func durtoTV(d time.Duration) (int64, int64) {
	sec := int64(d / nanoPerSec)
	micro := int64((int64(d) - sec*nanoPerSec) / 1000)

	return sec, micro
}

// GetTime returns the "receive time" from the remote NTP server
// specified as host.  NTP client mode is used.
func GetTime(host string) (msg, ntpTime, error) {
	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, "123"))
	if err != nil {
		fmt.Println(err)
		return msg{}, 0, err
	}

	con, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		fmt.Println(err)
		return msg{}, 0, err
	}

	defer con.Close()
	con.SetDeadline(time.Now().Add(5 * time.Second))

	m := new(msg)
	m.SetMode(client)
	m.SetVersion(4)
	m.TransmitTime = Encode(time.Now())

	err = binary.Write(con, binary.BigEndian, m)
	if err != nil {
		fmt.Println(err)
		return msg{}, 0, err
	}

	err = binary.Read(con, binary.BigEndian, m)
	if err != nil {
		fmt.Println(err)
		return msg{}, 0, err
	}

	dest := Encode(time.Now())
	if err != nil {
		fmt.Println(err)
		return msg{}, 0, err
	}

	return *m, dest, nil
}

// GetParams returns two time.Durations the time offset and rtt time
func GetParams(m msg, dest ntpTime) (offset time.Duration, rtt time.Duration) {
	T1 := m.OriginateTime
	T2 := m.ReceiveTime
	T3 := m.TransmitTime
	T4 := dest
	offset = (T2.Sub(T1) + T3.Sub(T4)) / 2
	rtt = T4.Sub(T1) - T2.Sub(T3)
	return
}

func GNTPClockRun(args []string) int {
	var servIP net.IP
	var saveClock bool
	switch len(args) {
	case 1:
		servIP = net.ParseIP(args[0])
		if servIP == nil {
			return 111
		}
	case 2:
		servIP = net.ParseIP(args[0])
		if servIP == nil {
			return 111
		}
		saveClock = args[1] == "saveclock"
	}

	m, dst, err := GetTime(servIP.String())
	if err != nil {
		fmt.Println(err)
		return 111
	}

	offset, rtt := GetParams(m, dst)

	fmt.Println("System time", time.Now())
	fmt.Println("Offset ", offset, "RTT ", rtt)

	if saveClock {
		t := time.Now().Add(offset)
		tv := syscall.NsecToTimeval(t.UnixNano())
		err := syscall.Settimeofday(&tv)

		if err != nil {
			fmt.Println(err)
			return 111
		}

	}
	return 0
}
