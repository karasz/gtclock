package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"syscall"
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

func checkErr(e error) {
	fmt.Println(e)
	os.Exit(111)
}

func tainExchange(m []byte, c *net.UDPConn) (answer []byte, t1 glibtai.TAIN) {
	answer = make([]byte, tainPacket)

	_, err := c.Write(m)
	if err != nil {
		checkErr(err)
	}
	t1 = glibtai.TAINNow()
	_, err = c.Read(answer)
	if err != nil {
		checkErr(err)
	}
	return answer, t1
}

func decodeResp(resp []byte) glibtai.TAIN {
	return glibtai.TAINUnpack(resp[4:16])
}

func dur(d time.Duration) (int64, int32) {
	seconds := d.Seconds()
	sec, nano := math.Modf(seconds)
	return int64(sec), int32(nano)
}

func main() {

	saveClock := flag.Bool("save", false, "update local clock")
	flag.Parse()

	servIP := net.ParseIP(flag.Arg(0))
	if servIP == nil {
		fmt.Println("Bad IP ")
		os.Exit(111)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(servIP.String(), "4014"))
	if err != nil {
		checkErr(err)
	}
	conn, err := net.DialUDP("udp", nil, serverAddr)

	if err != nil {
		checkErr(err)
	}
	defer conn.Close()

	var totalroundtrip time.Duration

	for i := 0; i < 10; i++ {

		q, t0 := makeQuery()

		_, t1 := tainExchange(q, conn)

		z, err := glibtai.TAINSub(t1, t0)
		if err == nil {
			totalroundtrip += z
		} else {
			fmt.Println(err)
		}

	}
	fmt.Println("before: ", glibtai.TAINTime(glibtai.TAINNow()))
	qf, _ := makeQuery()
	resp, _ := tainExchange(qf, conn)
	avgrtt := totalroundtrip / 20 // we have 10 roundtrips.
	serverSays := glibtai.TAINTime(decodeResp(resp)).Add(avgrtt)

	if *saveClock {
		tv := new(syscall.Timeval)
		z := serverSays.UnixNano()
		sec, nsec := dur(time.Duration(z))
		tv.Sec, tv.Usec = sec, int64(nsec*1000)
		err = syscall.Settimeofday(tv)
		if err != nil {
			checkErr(err)
		}

	}

	fmt.Println("after: ", glibtai.TAINTime(decodeResp(resp)).Add(avgrtt))
}
