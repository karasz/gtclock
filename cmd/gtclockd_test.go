package cmd

//revive:disable:cognitive-complexity
//revive:disable:cyclomatic
//revive:disable:function-length
import (
	"bytes"
	"flag"
	"net"
	"testing"
	"time"

	"github.com/karasz/gtclock/gtudpd"
)

func TestGTClockDRunFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{
			name:     "no arguments",
			args:     []string{},
			wantExit: 0, // Would normally run server, but we're only testing flag parsing
		},
		{
			name:     "valid config directory",
			args:     []string{"-d", "/tmp/test"},
			wantExit: 0,
		},
		{
			name:     "config directory with equals",
			args:     []string{"-d=/tmp/test"},
			wantExit: 0,
		},
		{
			name:     "invalid flag",
			args:     []string{"-x"},
			wantExit: 111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset configDir for each test
			oldConfigDir := configDir
			defer func() { configDir = oldConfigDir }()
			configDir = ""

			// Only test flag parsing, not actual server startup
			fs := flag.NewFlagSet("gtclockd", flag.ContinueOnError)
			fs.StringVar(&configDir, "d", "", "config directory path")

			err := fs.Parse(tt.args)
			if tt.wantExit == 111 && err == nil {
				t.Errorf("Expected flag parsing to fail, but it succeeded")
			}
			if tt.wantExit == 0 && err != nil {
				t.Errorf("Expected flag parsing to succeed, but got error: %v", err)
			}
		})
	}
}

func TestSendResponse(t *testing.T) {
	serverConn, clientConn := setupTestServer(t)
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	remoteAddr, _ := net.ResolveUDPAddr("udp", clientConn.LocalAddr().String())

	testBuf := make([]byte, 20)
	copy(testBuf[:4], []byte("ctai"))

	sendResponse(serverConn, 20, remoteAddr, testBuf)

	responseBuf := make([]byte, 256)
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := clientConn.Read(responseBuf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 20 {
		t.Fatalf("Response too short: got %d bytes, want at least 20", n)
	}

	if !bytes.Equal(responseBuf[:1], []byte("s")) {
		t.Errorf("Invalid response header: got %q, want %q", responseBuf[:1], "s")
	}

	// Verify TAIN timestamp is present in bytes 4-12
	if bytes.Equal(responseBuf[4:12], make([]byte, 8)) {
		t.Errorf("TAIN timestamp appears to be zero")
	}
}

func TestValidateTAINRequest(t *testing.T) {
	config := &gtudpd.Config{
		DefaultPort:    ":4014",
		ConfigDir:      "", // Empty means allow all IPs
		MaxRequestSize: 64, // Set the max request size for the test
	}

	validator := validateTAINRequest(config)
	testIP := net.ParseIP("127.0.0.1")

	tests := []struct {
		name string
		n    int
		buf  []byte
		want bool
	}{
		{
			name: "valid TAIN request",
			n:    20,
			buf:  append([]byte("ctai"), make([]byte, 16)...),
			want: true,
		},
		{
			name: "too short",
			n:    19,
			buf:  append([]byte("ctai"), make([]byte, 15)...),
			want: false,
		},
		{
			name: "invalid magic bytes",
			n:    20,
			buf:  append([]byte("xxxx"), make([]byte, 16)...),
			want: false,
		},
		{
			name: "too long",
			n:    65, // config.MaxRequestSize + 1 (default is 64)
			buf:  make([]byte, 65),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator(tt.n, tt.buf, testIP)
			if got != tt.want {
				t.Errorf("validateTAINRequest(%d, %q, %v) = %v, want %v",
					tt.n, tt.buf[:4], testIP, got, tt.want)
			}
		})
	}
}

// setupTestServer creates a pair of connected UDP sockets for testing
func setupTestServer(t *testing.T) (serverConn *net.UDPConn, clientConn *net.UDPConn) {
	// Create server socket
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverConn, err = net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	// Create client socket connected to server
	clientDialConn, err := net.Dial("udp", serverConn.LocalAddr().String())
	if err != nil {
		_ = serverConn.Close()
		t.Fatal(err)
	}

	var ok bool
	clientConn, ok = clientDialConn.(*net.UDPConn)
	if !ok {
		_ = serverConn.Close()
		t.Fatal("Failed to convert to UDP connection")
	}
	return serverConn, clientConn
}
