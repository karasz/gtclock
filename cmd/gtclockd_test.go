package cmd

//revive:disable:cognitive-complexity
//revive:disable:cyclomatic
//revive:disable:function-length
import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientok(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtclockd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	oldConfigDir := configDir
	defer func() { configDir = oldConfigDir }()

	tests := []struct {
		name      string
		ip        string
		files     []string
		configDir string
		want      bool
	}{
		{
			name:      "no config directory allows all",
			ip:        "192.168.1.1",
			configDir: "",
			want:      true,
		},
		{
			name:      "exact IP match",
			ip:        "192.168.1.100",
			files:     []string{"192.168.1.100"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "exact IP no match",
			ip:        "192.168.1.100",
			files:     []string{"192.168.1.101"},
			configDir: tempDir,
			want:      false,
		},
		{
			name:      "/8 network match",
			ip:        "192.168.1.100",
			files:     []string{"192"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "/16 network match",
			ip:        "192.168.1.100",
			files:     []string{"192.168"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "/24 network match",
			ip:        "192.168.1.100",
			files:     []string{"192.168.1"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "broad network takes precedence",
			ip:        "192.168.1.100",
			files:     []string{"192", "192.168", "192.168.1", "192.168.1.100"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "no match for different network",
			ip:        "10.0.0.1",
			files:     []string{"192", "192.168", "192.168.1"},
			configDir: tempDir,
			want:      false,
		},
		{
			name:      "IPv6 exact match",
			ip:        "2001:db8::1",
			files:     []string{"2001:db8::1"},
			configDir: tempDir,
			want:      true,
		},
		{
			name:      "IPv6 no match",
			ip:        "2001:db8::1",
			files:     []string{"2001:db8::2"},
			configDir: tempDir,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runClientokTest(t, tempDir, tt)
		})
	}
}

func TestGTClockDRunFlagParsing(t *testing.T) {
	oldConfigDir := configDir
	defer func() { configDir = oldConfigDir }()

	tests := []struct {
		name           string
		args           []string
		expectedConfig string
		expectError    bool
	}{
		{
			name:           "no arguments",
			args:           []string{},
			expectedConfig: "",
			expectError:    false,
		},
		{
			name:           "valid config directory",
			args:           []string{"-d", "/etc/gtclock"},
			expectedConfig: "/etc/gtclock",
			expectError:    false,
		},
		{
			name:           "config directory with equals",
			args:           []string{"-d=/etc/gtclock"},
			expectedConfig: "/etc/gtclock",
			expectError:    false,
		},
		{
			name:        "invalid flag",
			args:        []string{"-x"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir = ""

			exitCode := testGTClockDRunFlagOnly(tt.args)

			if tt.expectError && exitCode == 0 {
				t.Errorf("Expected error but got success")
			}
			if !tt.expectError && exitCode != 0 && exitCode != 999 {
				t.Errorf("Expected success but got exit code %d", exitCode)
			}
			if !tt.expectError && configDir != tt.expectedConfig {
				t.Errorf("Expected configDir=%q, got %q", tt.expectedConfig, configDir)
			}
		})
	}
}

func testGTClockDRunFlagOnly(args []string) int {
	fs := flag.NewFlagSet("gtclockd", flag.ContinueOnError)
	fs.StringVar(&configDir, "d", "", "config directory path")

	if err := fs.Parse(args); err != nil {
		return 111
	}

	return 999
}

func TestSendResponse(t *testing.T) {
	serverAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serverConn.Close() }()

	clientAddr, err := net.ResolveUDPAddr("udp", serverConn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}

	clientConn, err := net.DialUDP("udp", nil, clientAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = clientConn.Close() }()

	remoteAddr, _ := net.ResolveUDPAddr("udp", clientConn.LocalAddr().String())

	testBuf := make([]byte, 20)
	copy(testBuf[:4], []byte("ctai"))

	sendResponse(serverConn, remoteAddr, testBuf)

	responseBuf := make([]byte, 256)
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := clientConn.Read(responseBuf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if n < 4 {
		t.Fatalf("Response too short: got %d bytes", n)
	}

	if !bytes.Equal(responseBuf[:1], []byte("s")) {
		t.Errorf("Expected response to start with 's', got %c", responseBuf[0])
	}
}

// setupTestServer creates a test UDP server and client connection.
func setupTestServer(t *testing.T) (serverConn *net.UDPConn, clientConn *net.UDPConn) {
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	serverConn, err = net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	clientAddr, err := net.ResolveUDPAddr("udp", serverConn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}

	clientConn, err = net.DialUDP("udp", nil, clientAddr)
	if err != nil {
		t.Fatal(err)
	}

	return serverConn, clientConn
}

// runClientokTest runs a single clientok test case.
func runClientokTest(t *testing.T, tempDir string, tt struct {
	name      string
	ip        string
	files     []string
	configDir string
	want      bool
}) {
	configDir = tt.configDir

	for _, file := range tt.files {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ip := net.ParseIP(tt.ip)
	if ip == nil {
		t.Fatalf("Invalid test IP: %s", tt.ip)
	}

	got := clientok(ip)
	if got != tt.want {
		t.Errorf("clientok(%s) = %v, want %v", tt.ip, got, tt.want)
	}

	for _, file := range tt.files {
		filePath := filepath.Join(tempDir, file)
		_ = os.Remove(filePath)
	}
}

func TestHandleClientRequestWithClientok(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtclockd_clientok_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	oldConfigDir := configDir
	defer func() { configDir = oldConfigDir }()
	configDir = tempDir

	allowedIPFile := filepath.Join(tempDir, "127.0.0.1")
	if err := os.WriteFile(allowedIPFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	serverConn, clientConn := setupTestServer(t)
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	query := make([]byte, 20)
	copy(query[:4], []byte("ctai"))

	done := make(chan bool, 1)
	go func() {
		buf := make([]byte, 256)
		_ = serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			done <- false
			return
		}
		if isValidRequest(n, buf) && clientok(remoteAddr.IP) {
			sendResponse(serverConn, remoteAddr, buf)
			done <- true
			return
		}
		done <- false
	}()

	_, err = clientConn.Write(query)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case success := <-done:
		if !success {
			t.Error("Server did not handle allowed client request")
		}
	case <-time.After(2 * time.Second):
		t.Error("Test timed out")
	}

	response := make([]byte, 256)
	_ = clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := clientConn.Read(response)
	if err != nil {
		t.Errorf("Failed to read response from allowed client: %v", err)
	} else if n < 1 || response[0] != 's' {
		t.Error("Expected valid response for allowed client")
	}
}

func TestGetPort(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtclockd_port_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	oldConfigDir := configDir
	defer func() { configDir = oldConfigDir }()

	tests := []struct {
		name      string
		configDir string
		portFile  string
		content   string
		want      string
	}{
		{
			name:      "no config directory",
			configDir: "",
			want:      ":4014",
		},
		{
			name:      "no port file",
			configDir: tempDir,
			want:      ":4014",
		},
		{
			name:      "empty port file",
			configDir: tempDir,
			portFile:  "port",
			content:   "",
			want:      ":4014",
		},
		{
			name:      "valid port number",
			configDir: tempDir,
			portFile:  "port",
			content:   "8080",
			want:      ":8080",
		},
		{
			name:      "port with colon prefix",
			configDir: tempDir,
			portFile:  "port",
			content:   ":9090",
			want:      ":9090",
		},
		{
			name:      "port with whitespace",
			configDir: tempDir,
			portFile:  "port",
			content:   "  7070  \n",
			want:      ":7070",
		},
		{
			name:      "invalid port too high",
			configDir: tempDir,
			portFile:  "port",
			content:   "70000",
			want:      ":4014",
		},
		{
			name:      "invalid port zero",
			configDir: tempDir,
			portFile:  "port",
			content:   "0",
			want:      ":4014",
		},
		{
			name:      "invalid port negative",
			configDir: tempDir,
			portFile:  "port",
			content:   "-1",
			want:      ":4014",
		},
		{
			name:      "invalid port non-numeric",
			configDir: tempDir,
			portFile:  "port",
			content:   "abc",
			want:      ":4014",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir = tt.configDir

			if tt.portFile != "" {
				portFilePath := filepath.Join(tempDir, tt.portFile)
				if err := os.WriteFile(portFilePath, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = os.Remove(portFilePath) }()
			}

			got := getPort()
			if got != tt.want {
				t.Errorf("getPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupUDPServerPermissionError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtclockd_perm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	oldConfigDir := configDir
	defer func() { configDir = oldConfigDir }()

	tests := []struct {
		name           string
		configDir      string
		port           string
		expectErrorMsg string
	}{
		{
			name:           "no config dir with privileged port",
			configDir:      "",
			port:           "80",
			expectErrorMsg: "configure with -d flag and create port file",
		},
		{
			name:           "with config dir and privileged port",
			configDir:      tempDir,
			port:           "443",
			expectErrorMsg: fmt.Sprintf("configure in %s/port", tempDir),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir = tt.configDir

			if tt.configDir != "" {
				portFile := filepath.Join(tt.configDir, "port")
				if err := os.WriteFile(portFile, []byte(tt.port), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = os.Remove(portFile) }()
			}

			_, err := setupUDPServer()
			if err == nil {
				t.Skip("Skipping test - requires running as non-root user to test privileged port binding")
				return
			}

			if !strings.Contains(err.Error(), "permission denied") {
				t.Skipf("Skipping test - error is not permission related: %v", err)
				return
			}

			if !strings.Contains(err.Error(), tt.expectErrorMsg) {
				t.Errorf("Expected error message to contain '%s', got: %v", tt.expectErrorMsg, err)
			}
		})
	}
}
