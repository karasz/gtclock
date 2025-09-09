package cmd

//revive:disable:cognitive-complexity
import (
	"testing"
	"time"

	"github.com/karasz/glibtai"
)

func TestRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"single char", 1},
		{"small string", 8},
		{"medium string", 16},
		{"large string", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := randomString(tt.length)
			if len(result) != tt.length {
				t.Errorf("randomString(%d) = %q, want length %d, got %d",
					tt.length, result, tt.length, len(result))
			}

			// Check that all characters are from the allowed set
			for _, char := range result {
				if !containsChar(letterBytes, byte(char)) {
					t.Errorf("randomString(%d) contains invalid character %c", tt.length, char)
				}
			}
		})
	}

	// Test that multiple calls produce different results (probabilistic test)
	result1 := randomString(10)
	result2 := randomString(10)
	result3 := randomString(10)
	if result1 == result2 && result2 == result3 {
		t.Error("randomString appears to be deterministic - expected random output")
	}
}

func TestMakeQuery(t *testing.T) {
	query, _ := makeQuery()

	// Test query structure
	if len(query) != 28 {
		t.Errorf("makeQuery() query length = %d, want 28", len(query))
	}

	// Test magic bytes
	if string(query[0:4]) != "ctai" {
		t.Errorf("makeQuery() magic bytes = %q, want %q", string(query[0:4]), "ctai")
	}

	// Test that timestamp is packed correctly (check length and that it's not empty)
	if len(query[4:20]) != 16 {
		t.Errorf("makeQuery() timestamp section length = %d, want 16", len(query[4:20]))
	}

	// Verify timestamp section is not all zeros
	allZero := true
	for _, b := range query[4:20] {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("makeQuery() timestamp appears to be all zeros")
	}

	// Test that random suffix has correct length
	if len(query[20:28]) != 8 {
		t.Errorf("makeQuery() random suffix length = %d, want 8", len(query[20:28]))
	}
}

func TestDecodeResp(t *testing.T) {
	// Create a test response with known timestamp
	now := glibtai.TAINNow()
	packed := glibtai.TAINPack(now)

	resp := make([]byte, 28)
	copy(resp[0:4], []byte("resp")) // Magic bytes
	copy(resp[4:20], packed)        // Timestamp

	decoded := decodeResp(resp)

	// Compare seconds (allowing for small differences due to packing/unpacking)
	originalSec := glibtai.TAINTime(now).Unix()
	decodedSec := glibtai.TAINTime(decoded).Unix()

	if originalSec != decodedSec {
		t.Errorf("decodeResp() timestamp mismatch: original %d, decoded %d",
			originalSec, decodedSec)
	}
}

func TestDur(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantSec  int64
		wantFrac int32 // fractional part of seconds (0-1 range, not nanoseconds)
	}{
		{"zero", 0, 0, 0},
		{"one second", time.Second, 1, 0},
		{"half second", 500 * time.Millisecond, 0, 0},  // int32(0.5) == 0
		{"1.5 seconds", 1500 * time.Millisecond, 1, 0}, // int32(0.5) == 0
		{"negative", -time.Second, -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSec, gotFrac := dur(tt.duration)

			if gotSec != tt.wantSec {
				t.Errorf("dur(%v) sec = %d, want %d", tt.duration, gotSec, tt.wantSec)
			}

			// Check fractional part (note: int32 conversion truncates fractional values to 0)
			if gotFrac != tt.wantFrac {
				t.Errorf("dur(%v) frac = %d, want %d", tt.duration, gotFrac, tt.wantFrac)
			}
		})
	}
}

func TestParseGTClockArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantIP        string
		wantSaveClock bool
		wantErr       bool
	}{
		{"valid IP", []string{"192.168.1.1"}, "192.168.1.1", false, false},
		{"valid IP with saveclock", []string{"10.0.0.1", "saveclock"}, "10.0.0.1", true, false},
		{"valid IP with other arg", []string{"127.0.0.1", "other"}, "127.0.0.1", false, false},
		{"invalid IP", []string{"invalid"}, "", false, true},
		{"no arguments", []string{}, "", false, true},
		{"too many arguments", []string{"192.168.1.1", "arg1", "arg2"}, "", false, true},
		{"IPv6 address", []string{"2001:db8::1"}, "2001:db8::1", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotSaveClock, err := parseGTClockArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseGTClockArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotIP.String() != tt.wantIP {
					t.Errorf("parseGTClockArgs() IP = %v, want %v", gotIP, tt.wantIP)
				}
				if gotSaveClock != tt.wantSaveClock {
					t.Errorf("parseGTClockArgs() saveClock = %v, want %v", gotSaveClock, tt.wantSaveClock)
				}
			}
		})
	}
}

// Helper function to check if a byte is in the allowed character set
func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// Test the constants and package-level variables
func TestConstants(t *testing.T) {
	if tainPacket != 28 {
		t.Errorf("tainPacket = %d, want 28", tainPacket)
	}

	if letterIdxBits != 6 {
		t.Errorf("letterIdxBits = %d, want 6", letterIdxBits)
	}

	if letterIdxMask != 63 {
		t.Errorf("letterIdxMask = %d, want 63", letterIdxMask)
	}

	if len(letterBytes) != 52 {
		t.Errorf("letterBytes length = %d, want 52", len(letterBytes))
	}
}

// Benchmark tests for performance-sensitive functions
func BenchmarkRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		randomString(8)
	}
}

func BenchmarkMakeQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		makeQuery()
	}
}

func BenchmarkDur(b *testing.B) {
	d := 1500 * time.Millisecond
	for i := 0; i < b.N; i++ {
		dur(d)
	}
}
