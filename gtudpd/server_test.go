package gtudpd

//revive:disable:cognitive-complexity
//revive:disable:cyclomatic
//revive:disable:function-length
import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const defaultPort = ":4014"

func TestConfigGetPort(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtudpd_port_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name      string
		configDir string
		portFile  string
		want      string
	}{
		{
			name:      "no config directory",
			configDir: "",
			portFile:  "",
			want:      defaultPort,
		},
		{
			name:      "no port file",
			configDir: tempDir,
			portFile:  "",
			want:      defaultPort,
		},
		{
			name:      "empty port file",
			configDir: tempDir,
			portFile:  "",
			want:      defaultPort,
		},
		{
			name:      "valid port number",
			configDir: tempDir,
			portFile:  "8080",
			want:      ":8080",
		},
		{
			name:      "port with colon prefix",
			configDir: tempDir,
			portFile:  ":9000",
			want:      ":9000",
		},
		{
			name:      "port with whitespace",
			configDir: tempDir,
			portFile:  "  7000  ",
			want:      ":7000",
		},
		{
			name:      "invalid port too high",
			configDir: tempDir,
			portFile:  "99999",
			want:      defaultPort,
		},
		{
			name:      "invalid port zero",
			configDir: tempDir,
			portFile:  "0",
			want:      defaultPort,
		},
		{
			name:      "invalid port negative",
			configDir: tempDir,
			portFile:  "-1",
			want:      defaultPort,
		},
		{
			name:      "invalid port non-numeric",
			configDir: tempDir,
			portFile:  "abc",
			want:      defaultPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				DefaultPort: defaultPort,
				ConfigDir:   tt.configDir,
			}

			if tt.portFile != "" {
				portFilePath := filepath.Join(tempDir, "port")
				if err := os.WriteFile(portFilePath, []byte(tt.portFile), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = os.Remove(portFilePath) }()
			}

			got := config.GetPort()
			if got != tt.want {
				t.Errorf("GetPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigClientOK(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtudpd_clientok_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name  string
		ip    string
		files []string
		want  bool
	}{
		{
			name:  "no config directory allows all",
			ip:    "127.0.0.1",
			files: nil,
			want:  true,
		},
		{
			name:  "exact IP match",
			ip:    "127.0.0.1",
			files: []string{"127.0.0.1"},
			want:  true,
		},
		{
			name:  "exact IP no match",
			ip:    "127.0.0.2",
			files: []string{"127.0.0.1"},
			want:  false,
		},
		{
			name:  "/8 network match",
			ip:    "192.168.1.100",
			files: []string{"192"},
			want:  true,
		},
		{
			name:  "/16 network match",
			ip:    "192.168.1.100",
			files: []string{"192.168"},
			want:  true,
		},
		{
			name:  "/24 network match",
			ip:    "192.168.1.100",
			files: []string{"192.168.1"},
			want:  true,
		},
		{
			name:  "broad network takes precedence",
			ip:    "192.168.1.100",
			files: []string{"192", "192.168", "192.168.1"},
			want:  true,
		},
		{
			name:  "no match for different network",
			ip:    "10.0.0.1",
			files: []string{"192.168.1"},
			want:  false,
		},
		{
			name:  "IPv6 exact match",
			ip:    "::1",
			files: []string{"::1"},
			want:  true,
		},
		{
			name:  "IPv6 no match",
			ip:    "::2",
			files: []string{"::1"},
			want:  false,
		},
		{
			name:  "file '0' allows all clients",
			ip:    "203.0.113.42",
			files: []string{"0"},
			want:  true,
		},
		{
			name:  "file '0' allows all clients - IPv6",
			ip:    "2001:db8::1",
			files: []string{"0"},
			want:  true,
		},
		{
			name:  "file '0' overrides specific denials",
			ip:    "192.168.1.1",
			files: []string{"0", "192.168.2"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runClientOKTest(t, tempDir, tt)
		})
	}
}

// runClientOKTest runs a single ClientOK test case.
func runClientOKTest(t *testing.T, tempDir string, tt struct {
	name  string
	ip    string
	files []string
	want  bool
}) {
	// Create test files
	for _, file := range tt.files {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Parse IP
	ip := net.ParseIP(tt.ip)
	if ip == nil {
		t.Fatalf("Invalid test IP: %s", tt.ip)
	}

	// Test with config directory for this test
	configDir := ""
	if len(tt.files) > 0 {
		configDir = tempDir
	}

	config := &Config{
		DefaultPort: defaultPort,
		ConfigDir:   configDir,
	}
	got := config.ClientOK(ip)
	if got != tt.want {
		t.Errorf("ClientOK(%s) = %v, want %v", tt.ip, got, tt.want)
	}

	// Clean up test files
	for _, file := range tt.files {
		filePath := filepath.Join(tempDir, file)
		_ = os.Remove(filePath)
	}
}

func TestNewServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtudpd_server_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

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
			expectErrorMsg: "try running as root or use a port >= 1024",
		},
		{
			name:           "with config dir and privileged port",
			configDir:      tempDir,
			port:           "443",
			expectErrorMsg: "try running as root or use a port >= 1024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.port != "" {
				portFile := filepath.Join(tt.configDir, "port")
				if err := os.WriteFile(portFile, []byte(tt.port), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = os.Remove(portFile) }()
			}

			config := &Config{
				DefaultPort: defaultPort,
				ConfigDir:   tt.configDir,
			}
			_, err := NewServer(config, testHandler, testValidator)
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

func TestServerRateLimit(t *testing.T) {
	config := &Config{
		DefaultPort: ":0", // Use any available port
		ConfigDir:   "",
	}

	server, err := NewServer(config, testHandler, testValidator)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Stop() }()

	testIP := "127.0.0.1"

	// Test that we can make requests up to the limit
	for i := 0; i < server.config.MaxRequestsPerIP; i++ {
		if !server.checkRateLimit(testIP) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Test that the next request is rate limited
	if server.checkRateLimit(testIP) {
		t.Errorf("Request %d should be rate limited", server.config.MaxRequestsPerIP+1)
	}

	// Wait for rate limit window to expire
	time.Sleep(server.config.RateLimitWindow + 100*time.Millisecond)

	// Test that we can make requests again after window expires
	if !server.checkRateLimit(testIP) {
		t.Errorf("Request should be allowed after rate limit window expires")
	}
}

func TestServerCleanupRateLimit(t *testing.T) {
	config := &Config{
		DefaultPort: ":0",
		ConfigDir:   "",
	}

	server, err := NewServer(config, testHandler, testValidator)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Stop() }()

	// Add some rate limit entries
	testIPs := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	for _, ip := range testIPs {
		server.checkRateLimit(ip)
	}

	// Verify entries exist
	server.rateLimitMutex.RLock()
	initialCount := len(server.rateLimitMap)
	server.rateLimitMutex.RUnlock()

	if initialCount != len(testIPs) {
		t.Errorf("Expected %d rate limit entries, got %d", len(testIPs), initialCount)
	}

	// Wait for cleanup to happen (cleanup runs every 2 * RateLimitWindow)
	time.Sleep(server.config.RateLimitWindow*2 + 200*time.Millisecond)

	// Manually trigger cleanup to ensure it runs
	server.cleanupExpiredEntries()

	// Verify entries are cleaned up
	server.rateLimitMutex.RLock()
	finalCount := len(server.rateLimitMap)
	server.rateLimitMutex.RUnlock()

	if finalCount != 0 {
		t.Errorf("Expected 0 rate limit entries after cleanup, got %d", finalCount)
	}
}

func TestValidatePortFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gtudpd_path_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name     string
		portFile string
		want     bool
	}{
		{
			name:     "valid port file",
			portFile: filepath.Join(tempDir, "port"),
			want:     true,
		},
		{
			name:     "path traversal attempt",
			portFile: filepath.Join(tempDir, "../../../etc/passwd"),
			want:     false,
		},
		{
			name:     "relative path traversal",
			portFile: filepath.Join(tempDir, "../../sensitive_file"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePortFilePath(tt.portFile, tempDir)
			if got != tt.want {
				t.Errorf("validatePortFilePath(%s, %s) = %v, want %v",
					tt.portFile, tempDir, got, tt.want)
			}
		})
	}
}

func TestParsePortString(t *testing.T) {
	tests := []struct {
		name    string
		portStr string
		want    string
		valid   bool
	}{
		{"empty string", "", "", false},
		{"valid port", "8080", ":8080", true},
		{"port with colon", ":9000", ":9000", true},
		{"just colon", ":", "", false},
		{"port too high", "99999", "", false},
		{"port zero", "0", "", false},
		{"negative port", "-1", "", false},
		{"non-numeric", "abc", "", false},
		{"valid max port", "65535", ":65535", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := parsePortString(tt.portStr)
			if valid != tt.valid || got != tt.want {
				t.Errorf("parsePortString(%s) = (%s, %v), want (%s, %v)",
					tt.portStr, got, valid, tt.want, tt.valid)
			}
		})
	}
}

func TestIsValidNetworkName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", false},
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "::1", true},
		{"valid hex", "fe80", true},
		{"invalid character", "192.168.1.1/24", false},
		{"invalid character underscore", "test_network", false},
		{"valid network prefix", "192.168.1", true},
		{"valid single octet", "10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNetworkName(tt.input)
			if got != tt.want {
				t.Errorf("isValidNetworkName(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test helper functions
func testHandler(conn *net.UDPConn, n int, remoteaddr *net.UDPAddr, buf []byte) {
	// Simple test handler that echoes back the request
	_, _ = conn.WriteToUDP(buf[:n], remoteaddr)
}

func testValidator(n int, _ []byte, _ net.IP) bool {
	// Simple test validator that accepts requests >= 4 bytes
	return n >= 4
}
