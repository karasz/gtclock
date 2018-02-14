// Copyright © 2018 Nagy Károly Gábriel <karasz@jpi.io>
// This file, part of glibtai, is free and unencumbered software
// released into the public domain.
// For more information, please refer to <http://unlicense.org/>

// +build linux,amd64

// Package glibtai is a partial Go implementation of libtai. See
// http://cr.yp.to/libtai/ for more information.
package glibtai

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

// TAI struct to store a TAI timestamp
type TAI struct {
	x uint64
}

// TAIN struct to store TAIN timestamps
type TAIN struct {
	sec  uint64
	nano uint32
}

// TAICONST is 2^62+10 representing the TAI label of the second Unix started
// 1970-01-01 00:00:00 +0000 UTC
const TAICONST = uint64(4611686018427387914)

// TAILength is the length of a TAI timestamp in bytes
const TAILength = 8

// TAINLength is the length of a TAIN timestamp in bytes
const TAINLength = 12

// TAINow returns the current timestamp in TAI struct
func TAINow() TAI {
	return TAI{x: TAICONST + lsoffset(time.Now()) + uint64(time.Now().Unix())}
}

// TAINNow returns the current timestamp in TAIN struct
func TAINNow() TAIN {
	now := time.Now()
	return TAIN{
		sec:  TAICONST + lsoffset(now) + uint64(now.Unix()),
		nano: uint32(now.Nanosecond()),
	}
}

//TAIAdd  adds a time.Duration to a TAI timestamp
func TAIAdd(a TAI, b time.Duration) TAI {
	return TAI{x: a.x + uint64(b.Seconds())}
}

//TAINAdd adds a time.Duration to a TAIN timestamp
func TAINAdd(a TAIN, b time.Duration) TAIN {
	var result TAIN
	result.sec = a.sec + uint64(b.Seconds())
	result.nano = a.nano + uint32(b.Nanoseconds()-int64(b.Seconds())*1000000000)
	if result.nano > 999999999 {
		result.sec++
		result.nano -= 1000000000
	}
	return result
}

// TAISub subtracts two TAI timestamps
func TAISub(a, b TAI) (time.Duration, error) {
	x := a.x - b.x
	q, err := time.ParseDuration(fmt.Sprintf("%ds", x))
	return q, err
}

// TAINSub subtracts two TAI timestamps
func TAINSub(a, b TAIN) (time.Duration, error) {
	s := a.sec - b.sec
	n := a.nano - b.nano
	if n > a.nano {
		n += 1000000000
		s--
	}
	q, err := time.ParseDuration(fmt.Sprintf("%ds%dns", s, n))
	return q, err
}

// TAITime returns a go time object from a TAI timestamp
func TAITime(t TAI) time.Time {
	tm := time.Unix(int64(t.x-TAICONST), 0).UTC()
	return tm.Add(-time.Duration(lsoffset(tm)) * time.Second)
}

// TAINTime returns a go time object from a TAIN timestamp
func TAINTime(t TAIN) time.Time {
	tm := time.Unix(int64(t.sec-TAICONST), int64(t.nano)).UTC()
	return tm.Add(-time.Duration(lsoffset(tm)) * time.Second)
}

// TAIPack packs a TAI timestamp into a byte array of size TAILength
func TAIPack(t TAI) []byte {
	result := make([]byte, TAILength)
	binary.BigEndian.PutUint64(result[:], t.x)
	return result
}

// TAIUnpack unpacks a TAI timestamp from a byte array of size TAILength
func TAIUnpack(s []byte) TAI {
	return TAI{x: binary.BigEndian.Uint64(s[:])}
}

// TAINPack packs a TAIN timestamp in a byte array of size TAINLength
func TAINPack(t TAIN) []byte {
	result := make([]byte, TAINLength)
	binary.BigEndian.PutUint64(result[:], t.sec)
	binary.BigEndian.PutUint32(result[TAILength:], t.nano)
	return result
}

// TAINUnpack unpacks a TAIN timestamp from a byte array of size TAINLength
func TAINUnpack(s []byte) TAIN {
	var result TAIN
	result.sec = binary.BigEndian.Uint64(s[:])
	result.nano = binary.BigEndian.Uint32(s[TAILength:])
	return result
}

func (t TAI) String() string {
	buf := TAIPack(t)
	s := fmt.Sprintf("@%02X%02X%02X%02X%02X%02X%02X%02X",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6],
		buf[7])
	return s
}

func (t TAIN) String() string {
	buf := TAINPack(t)
	s := fmt.Sprintf("@%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6],
		buf[7], buf[8], buf[9], buf[10], buf[11])
	return s
}

// TAIfromString returns a TAI struct from an ASCII TAI representation
func TAIfromString(str string) (TAI, error) {
	if str[0] != '@' {
		return TAI{}, fmt.Errorf("TAI representation  %s is not valid, it does not begin with an '@'", str)
	}

	buf, err := hex.DecodeString(str[1:])
	if len(buf) != TAILength || err != nil {
		return TAI{}, err
	}

	return TAIUnpack(buf[:]), nil
}

//TAIfromTime returns a TAI struct from time.Time
func TAIfromTime(t time.Time) TAI {
	return TAI{x: TAICONST + lsoffset(t) + uint64(t.Unix())}
}

// TAINfromString returns a TAIN struct from an ASCII TAIN representation
func TAINfromString(str string) (TAIN, error) {
	if str[0] != '@' {
		return TAIN{}, fmt.Errorf("TAI representation  %s is not valid, it does not begin with an '@'", str)
	}

	buf, err := hex.DecodeString(str[1:])
	if len(buf) != TAINLength || err != nil {
		return TAIN{}, err
	}

	return TAINUnpack(buf[:]), nil
}

//TAINfromTime returns a TAIN struct from time.Time
func TAINfromTime(t time.Time) TAIN {
	return TAIN{
		sec:  TAICONST + lsoffset(t) + uint64(t.Unix()),
		nano: uint32(t.Nanosecond())}
}
