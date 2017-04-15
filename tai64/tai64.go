package tai64

import (
	"encoding/binary"
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
	binary.BigEndian.PutUint64(result[:], t.x)
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

// TaiSub subtracts two TAI timestamps
func TaiSub(a, b Tai) Tai {
	var result Tai
	result.x = a.x - b.x
	return result
}

// TainSub subtracts two TAI timestamps
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
	result.x = binary.BigEndian.Uint64(s[:])
	return result
}

// TainPack packs a TAIN timestamp in a byte slice
func TainPack(t Tain) []byte {
	result := make([]byte, TainCount)
	binary.BigEndian.PutUint64(result[:], t.sec.x)
	binary.BigEndian.PutUint32(result[8:], t.nano)
	return result
}

// TainUnpack unpacks a TAIN timestamp from a byte slice
func TainUnpack(s []byte) Tain {
	var result Tain
	result.sec.x = binary.BigEndian.Uint64(s[:])
	result.nano = binary.BigEndian.Uint32(s[8:])
	return result
}
