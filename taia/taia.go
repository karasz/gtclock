package tai64

import (
	"fmt"
	"syscall"
	"time"
)

type Tai struct {
	x uint64
}

type Taia struct {
	sec  Tai
	nano uint64
	atto uint64
}

const TAICONST = 4611686018427387914
const Tai_Count = 8
const Taia_Count = 16

func Tai_now() Tai {
	var result Tai
	result.x = TAICONST + uint64(time.Now().Unix())
	return result
}

func Taia_now() Taia {
	var result Taia
	now := new(syscall.Timeval)
	err := syscall.Gettimeofday(now)
	if err == nil {
		var t Tai
		t.x = TAICONST + uint64(now.Sec)
		result.sec = t
		result.nano = uint64(1000*uint64(now.Usec) + 500)
		result.atto = 0
	} else {
		fmt.Println(err)
	}
	return result
}

func Tai_pack(t Tai) []byte {
	result := make([]byte, Tai_Count)
	x := t.x
	result[7] = byte(x & 255)
	x >>= 8
	result[6] = byte(x & 255)
	x >>= 8
	result[5] = byte(x & 255)
	x >>= 8
	result[4] = byte(x & 255)
	x >>= 8
	result[3] = byte(x & 255)
	x >>= 8
	result[2] = byte(x & 255)
	x >>= 8
	result[1] = byte(x & 255)
	x >>= 8
	result[0] = byte(x)
	return result

}

func Tai_unpack(s []byte) Tai {
	var result Tai
	var x uint64
	x = uint64(s[0])
	x <<= 8
	x += uint64(s[1])
	x <<= 8
	x += uint64(s[2])
	x <<= 8
	x += uint64(s[3])
	x <<= 8
	x += uint64(s[4])
	x <<= 8
	x += uint64(s[5])
	x <<= 8
	x += uint64(s[6])
	x <<= 8
	x += uint64(s[7])
	result.x = x
	return result
}

func Taia_pack(t Taia) []byte {
	result := make([]byte, Taia_Count)
	zz := make([]byte, Tai_Count)
	zz = Tai_pack(t.sec)
	for i := 0; i < Tai_Count; i++ {
		result[i+Tai_Count] = zz[i]
	}
	x := t.atto
	result[7] = byte(x & 255)
	x >>= 8
	result[6] = byte(x & 255)
	x >>= 8
	result[5] = byte(x & 255)
	x >>= 8
	result[4] = byte(x)

	x = t.nano
	result[3] = byte(x & 255)
	x >>= 8
	result[2] = byte(x & 255)
	x >>= 8
	result[1] = byte(x & 255)
	x >>= 8
	result[0] = byte(x)

	return result
}

func Taia_unpack(s []byte) Taia {
	var result Taia
	var zz Tai
	zz = Tai_unpack(s[8:])
	result.sec = zz
	x := uint64(s[4])
	x <<= 8
	x += uint64(s[5])
	x <<= 8
	x += uint64(s[6])
	x <<= 8
	x += uint64(s[7])
	result.atto = x
	x = uint64(s[0])
	x <<= 8
	x += uint64(s[1])
	x <<= 8
	x += uint64(s[2])
	x <<= 8
	x += uint64(s[3])
	result.nano = x
	return result

}
