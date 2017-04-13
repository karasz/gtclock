package tai64

import (
	"fmt"
	"syscall"
	"time"
)

// Tai struct to store a TAI timestamp
type Tai struct {
	x uint64
}

// Tain struct to store TAIN timestamps
type Tain struct {
	sec  Tai
	nano uint64
}

// TAICONST represents the second TAI started
const TAICONST = 4611686018427387914

// TaiCount is the length of a TAI timestamp
const TaiCount = 8

// TaiaCount is the length of a TAIN timestamp
const TainCount = 12

// TaiNow returns the current time in TAI format
func TaiNow() Tai {
	var result Tai
	result.x = TAICONST + uint64(time.Now().Unix())
	return result
}

// TainNow returns the current time in TAIN format
func TainNow() Tain {
	var result Tain
	now := new(syscall.Timeval)
	err := syscall.Gettimeofday(now)
	if err == nil {
		var t Tai
		t.x = TAICONST + uint64(now.Sec)
		result.sec = t
		result.nano = uint64(1000*uint64(now.Usec) + 500)
	} else {
		fmt.Println(err)
	}
	return result
}

// TaiPack packs a TAI timestamp in a byte slice
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

// TaiPack unpacks a TAI timestamp from a byte slice
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

// TainPack packs a TAIN timestamp in a byte slice
func TainPack(t Tain) []byte {
	result := make([]byte, TainCount)
	zz := make([]byte, TaiCount)
	zz = TaiPack(t.sec)

	for i := 0; i < TaiCount; i++ {
		result[i+4] = zz[i]
	}
	x := t.nano
	result[3] = byte(x & 255)
	x >>= 8
	result[2] = byte(x & 255)
	x >>= 8
	result[1] = byte(x & 255)
	x >>= 8
	result[0] = byte(x)

	return result
}

// TainPack unpacks a TAIN timestamp from a byte slice
func TainUnpack(s []byte) Tain {
	var result Tain
	var zz Tai
	zz = TaiUnpack(s[8:])
	result.sec = zz
	x := uint64(s[0])
	x <<= 8
	x += uint64(s[1])
	x <<= 8
	x += uint64(s[2])
	x <<= 8
	x += uint64(s[3])
	result.nano = x
	return result

}
