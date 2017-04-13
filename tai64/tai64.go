package tai64

import (
	"fmt"
	"syscall"
	"time"
)

// Tai represents the second TAI started
type Tai struct {
	x uint64
}

// Taia struct to store taia
type Taia struct {
	sec  Tai
	nano uint64
	atto uint64
}

// TAICONST represents the second TAI started
const TAICONST = 4611686018427387914

// TaiCount is the length of a Tai timestamp
const TaiCount = 8

// TaiaCount is the length of a Taia timestamp
const TaiaCount = 16

func TaiNow() Tai {
	var result Tai
	result.x = TAICONST + uint64(time.Now().Unix())
	return result
}

func TaiaNow() Taia {
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

func TaiPack(t Tai) []byte {
	result := make([]byte, TaiCount)
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

func TaiUnpack(s []byte) Tai {
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

func TaiaPack(t Taia) []byte {
	result := make([]byte, TaiaCount)
	zz := make([]byte, TaiCount)
	zz = TaiPack(t.sec)
	for i := 0; i < TaiCount; i++ {
		result[i+TaiCount] = zz[i]
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

func TaiaUnpack(s []byte) Taia {
	var result Taia
	var zz Tai
	zz = TaiUnpack(s[8:])
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
