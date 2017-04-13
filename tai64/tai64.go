package tai64

import (
	"time"
)

// Tai struct to store a TAI timestamp
type Tai struct {
	x uint64
}

// Tain struct to store TAIN timestamps
type Tain struct {
	sec  Tai
	nano uint32
}

// TAICONST represents the second TAI started
const TAICONST = 4611686018427387914

// TaiCount is the length of a TAI timestamp
const TaiCount = 8

// TainCount is the length of a TAIN timestamp
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
	now := time.Now()
	var t Tai
	t.x = TAICONST + uint64(now.Unix())
	result.sec = t
	result.nano = uint32(now.Nanosecond())
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

//TaiAdd computes the sum of two TAI timestamps
func TaiAdd(a, b Tai) Tai {
	var result Tai
	result.x = a.x + b.x
	return result
}

//TainAdd computes the sum of two TAIN timestamps
func TainAdd(a, b Tain) Tain {
	var result Tain
	result.sec.x = a.sec.x + b.sec.x
	result.nano = a.nano + b.nano
	if result.nano > 999999999 {
		result.sec.x++
		result.nano -= 1000000000
	}
	return result
}

// TaiSub substracts two TAI timestamps
func TaiSub(a, b Tai) Tai {
	var result Tai
	result.x = a.x - b.x
	return result
}

// TainSub substracts two TAI timestamps
func TainSub(a, b Tain) Tain {
	var result Tain
	result.sec.x = a.sec.x - b.sec.x
	result.nano = a.nano - b.nano
	if result.nano > a.nano {
		result.nano += 1000000000
		result.sec.x--
	}
	return result
}

// TaiTime returns a go time object from a TAI timestamp
func TaiTime(t Tai) (time.Time, error) {
	var result time.Time
	result = time.Unix(int64(t.x-TAICONST), 0)
	return result, nil
}

// TainTime returns a go time object from a TAIN timestamp
func TainTime(t Tain) (time.Time, error) {
	var result time.Time
	result = time.Unix(int64(t.sec.x-TAICONST), int64(t.nano))
	return result, nil
}

// TaiUnpack unpacks a TAI timestamp from a byte slice
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

// TainUnpack unpacks a TAIN timestamp from a byte slice
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
	result.nano = uint32(x)
	return result

}
