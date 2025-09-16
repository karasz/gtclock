package cmd

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/karasz/glibtai"
)

const tainPacket = 28

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

func randomString(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func makeQuery() (query []byte, t0 glibtai.TAIN) {
	query = make([]byte, 28)
	e := []byte("ctai")
	copy(query[0:], e)
	t0 = glibtai.TAINNow()
	t := glibtai.TAINPack(t0)
	copy(query[4:], t)
	z := []byte(randomString(8))
	copy(query[20:], z)
	return query, t0
}

func tainExchange(m []byte, c *net.UDPConn) (answer []byte, t1 glibtai.TAIN, e error) {
	answer = make([]byte, tainPacket)

	_, err := c.Write(m)
	if err != nil {
		_, _ = fmt.Println(err)
		return answer, glibtai.TAIN{}, err
	}
	t1 = glibtai.TAINNow()
	_, err = c.Read(answer)
	if err != nil {
		_, _ = fmt.Println(err)
		return answer, glibtai.TAIN{}, err
	}
	return answer, t1, nil
}

func decodeResp(resp []byte) glibtai.TAIN {
	return glibtai.TAINUnpack(resp[4:16])
}

func dur(d time.Duration) (int64, int32) {
	seconds := d.Seconds()
	sec, nano := math.Modf(seconds)
	return int64(sec), int32(nano)
}

// parseGTClockArgs parses command line arguments for GTClock client.
func parseGTClockArgs(args []string) (servIP net.IP, saveClock bool, err error) {
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
		return nil, false, errors.New("unknown number of arguments. Please use 1 or 2")
	}
	return servIP, saveClock, nil
}

// measureServerTime performs time synchronization measurements with the server.
func measureServerTime(conn *net.UDPConn) (time.Time, error) {
	var totalroundtrip time.Duration

	for i := 0; i < 10; i++ {
		q, t0 := makeQuery()

		_, t1, e := tainExchange(q, conn)
		if e != nil {
			_, _ = fmt.Println(e)
		}

		z, err := glibtai.TAINSub(t1, t0)
		if err == nil {
			totalroundtrip += z
		} else {
			_, _ = fmt.Println(err)
		}
	}
	_, _ = fmt.Println("before: ", glibtai.TAINTime(glibtai.TAINNow()))
	qf, _ := makeQuery()
	resp, _, e := tainExchange(qf, conn)
	if e != nil {
		return time.Time{}, e
	}

	avgrtt := totalroundtrip / 20 // we have 10 roundtrips.
	serverSays := glibtai.TAINTime(decodeResp(resp)).Add(avgrtt)
	return serverSays, nil
}

// GTClockCRun implements the gtclockc client functionality for TAIN time synchronization.
func GTClockCRun(args []string) int {
	servIP, saveClock, err := parseGTClockArgs(args)
	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}
	serverAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(servIP.String(), "4014"))
	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}
	conn, err := net.DialUDP("udp", nil, serverAddr)

	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}
	defer func() { _ = conn.Close() }()

	serverSays, err := measureServerTime(conn)
	if err != nil {
		_, _ = fmt.Println(err)
		return 111
	}

	if saveClock {
		err = setSystemClockTime(serverSays)
		if err != nil {
			_, _ = fmt.Println(err)
			return 111
		}
	}
	_, _ = fmt.Println("after: ", serverSays)
	return 0
}
