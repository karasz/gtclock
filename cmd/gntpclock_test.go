package cmd

//revive:disable:cognitive-complexity
import (
	"math"
	"testing"
	"time"
)

// Test constants and types
func TestNTPConstants(t *testing.T) {
	if nanoPerSec != 1e9 {
		t.Errorf("nanoPerSec = %g, want 1000000000", nanoPerSec)
	}

	expectedEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	if !ntpEpoch.Equal(expectedEpoch) {
		t.Errorf("ntpEpoch = %v, want %v", ntpEpoch, expectedEpoch)
	}
}

func TestModeConstants(t *testing.T) {
	if reserved != 0 {
		t.Errorf("reserved = %d, want 0", reserved)
	}
	if client != 3 {
		t.Errorf("client = %d, want 3", client)
	}
	if server != 4 {
		t.Errorf("server = %d, want 4", server)
	}
}

// Test ntpTime methods
func TestNTPTimeDuration(t *testing.T) {
	tests := []struct {
		name     string
		ntpTime  ntpTime
		expected time.Duration
	}{
		{"zero", 0, 0},
		{"one second", 1 << 32, time.Second},
		{"half second", 1 << 31, 500 * time.Millisecond},
		{"two seconds", 2 << 32, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ntpTime.duration()
			// Allow small tolerance for floating point precision
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Microsecond {
				t.Errorf("ntpTime(%d).duration() = %v, want %v",
					uint64(tt.ntpTime), result, tt.expected)
			}
		})
	}
}

func TestNTPTimeDecode(t *testing.T) {
	// Test decoding of known NTP timestamp
	now := time.Now().UTC()
	encoded := encode(now)
	decoded := encoded.decode()

	// Allow 1 second tolerance due to precision loss in NTP format
	diff := decoded.Sub(now)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("decode/encode roundtrip error too large: %v", diff)
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
	}{
		{"epoch", time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"unix epoch", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"recent time", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encode(tt.time)
			decoded := encoded.decode()

			// Check that encode/decode is reasonably accurate (within 1 second)
			diff := decoded.Sub(tt.time)
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("encode/decode roundtrip failed for %v: got %v, diff %v",
					tt.time, decoded, diff)
			}
		})
	}
}

func TestNTPTimeSub(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	later := base.Add(5 * time.Second)

	ntpBase := encode(base)
	ntpLater := encode(later)

	diff := ntpLater.sub(ntpBase)
	expected := 5 * time.Second

	// Allow some tolerance for precision
	if math.Abs(float64(diff-expected)) > float64(100*time.Millisecond) {
		t.Errorf("ntpTime.sub() = %v, want %v", diff, expected)
	}
}

// Test msg methods
func TestMsgSetVersion(t *testing.T) {
	m := &msg{}

	// Test setting version 4
	m.setVersion(4)
	version := (m.LiVnMode >> 3) & 0x7
	if version != 4 {
		t.Errorf("setVersion(4): got version %d, want 4", version)
	}

	// Test setting version 3
	m.setVersion(3)
	version = (m.LiVnMode >> 3) & 0x7
	if version != 3 {
		t.Errorf("setVersion(3): got version %d, want 3", version)
	}
}

func TestMsgSetMode(t *testing.T) {
	m := &msg{}

	// Test setting client mode
	m.setMode(client)
	mode := m.LiVnMode & 0x7
	if mode != byte(client) {
		t.Errorf("setMode(client): got mode %d, want %d", mode, client)
	}

	// Test setting server mode
	m.setMode(server)
	mode = m.LiVnMode & 0x7
	if mode != byte(server) {
		t.Errorf("setMode(server): got mode %d, want %d", mode, server)
	}
}

func TestMsgVersionAndMode(t *testing.T) {
	m := &msg{}

	// Test that setting version doesn't affect mode and vice versa
	m.setVersion(4)
	m.setMode(client)

	version := (m.LiVnMode >> 3) & 0x7
	mode := m.LiVnMode & 0x7

	if version != 4 {
		t.Errorf("version after setting both: got %d, want 4", version)
	}
	if mode != byte(client) {
		t.Errorf("mode after setting both: got %d, want %d", mode, client)
	}
}

// Test utility functions
func TestGetParams(t *testing.T) {
	// Create a mock NTP message with known timestamps
	now := time.Now()

	m := msg{
		OriginateTime: encode(now),                            // T1
		ReceiveTime:   encode(now.Add(10 * time.Millisecond)), // T2
		TransmitTime:  encode(now.Add(20 * time.Millisecond)), // T3
	}
	dest := encode(now.Add(30 * time.Millisecond)) // T4

	offset, rtt := getParams(m, dest)

	// With these values:
	// offset = ((T2-T1) + (T3-T4))/2 = ((10ms) + (-10ms))/2 = 0
	// rtt = (T4-T1) - (T2-T3) = 30ms - (-10ms) = 40ms

	expectedOffset := time.Duration(0)
	expectedRTT := 40 * time.Millisecond

	// Allow some tolerance for precision
	if math.Abs(float64(offset-expectedOffset)) > float64(time.Millisecond) {
		t.Errorf("getParams() offset = %v, want ~%v", offset, expectedOffset)
	}

	if math.Abs(float64(rtt-expectedRTT)) > float64(5*time.Millisecond) {
		t.Errorf("getParams() rtt = %v, want ~%v", rtt, expectedRTT)
	}
}

func TestParseNTPArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantIP        string
		wantSaveClock bool
		wantErr       bool
	}{
		{"no args", []string{}, "", false, true},
		{"valid IPv4", []string{"192.168.1.1"}, "192.168.1.1", false, false},
		{"valid IPv4 with saveclock", []string{"10.0.0.1", "saveclock"}, "10.0.0.1", true, false},
		{"valid IPv4 with other arg", []string{"127.0.0.1", "other"}, "127.0.0.1", false, false},
		{"valid IPv6", []string{"2001:db8::1"}, "2001:db8::1", false, false},
		{"invalid IP", []string{"not-an-ip"}, "", false, true},
		{"too many args", []string{"1.1.1.1", "arg1", "arg2"}, "", false, true},
		{"empty IP", []string{""}, "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotSaveClock, err := parseNTPArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseNTPArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotIP.String() != tt.wantIP {
					t.Errorf("parseNTPArgs() IP = %v, want %v", gotIP, tt.wantIP)
				}
				if gotSaveClock != tt.wantSaveClock {
					t.Errorf("parseNTPArgs() saveClock = %v, want %v", gotSaveClock, tt.wantSaveClock)
				}
			}
		})
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkEncode(b *testing.B) {
	now := time.Now()
	for i := 0; i < b.N; i++ {
		encode(now)
	}
}

func BenchmarkNTPTimeDecode(b *testing.B) {
	nt := encode(time.Now())
	for i := 0; i < b.N; i++ {
		nt.decode()
	}
}

func BenchmarkNTPTimeDuration(b *testing.B) {
	nt := ntpTime(1<<32 + 1<<16) // 1.something seconds
	for i := 0; i < b.N; i++ {
		nt.duration()
	}
}

func BenchmarkGetParams(b *testing.B) {
	now := time.Now()
	m := msg{
		OriginateTime: encode(now),
		ReceiveTime:   encode(now.Add(10 * time.Millisecond)),
		TransmitTime:  encode(now.Add(20 * time.Millisecond)),
	}
	dest := encode(now.Add(30 * time.Millisecond))

	for i := 0; i < b.N; i++ {
		getParams(m, dest)
	}
}

// Test edge cases and error conditions
func TestNTPTimeEdgeCases(t *testing.T) {
	// Test maximum ntpTime value
	maxNTP := ntpTime(^uint64(0)) // Max uint64
	duration := maxNTP.duration()
	if duration < 0 {
		t.Error("maximum ntpTime should not produce negative duration")
	}

	// Test zero ntpTime
	zero := ntpTime(0)
	if zero.duration() != 0 {
		t.Error("zero ntpTime should produce zero duration")
	}
}

func TestEncodeEdgeCases(t *testing.T) {
	// Test time before NTP epoch (should handle gracefully)
	beforeEpoch := time.Date(1899, 12, 31, 23, 59, 59, 0, time.UTC)
	encoded := encode(beforeEpoch)

	// Should not panic and should produce some reasonable result
	decoded := encoded.decode()
	_ = decoded // Just ensure it doesn't panic

	// Test near future time (NTP timestamps have limited range)
	nearFuture := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	encodedFuture := encode(nearFuture)
	decodedFuture := encodedFuture.decode()

	// Should be reasonably close (within a few seconds due to precision)
	diff := decodedFuture.Sub(nearFuture)
	if diff < 0 {
		diff = -diff
	}
	if diff > 10*time.Second {
		t.Errorf("near future encode/decode diff too large: %v", diff)
	}
}
