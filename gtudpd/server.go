// Package gtudpd provides a secure UDP server implementation with rate limiting,
// timeout protection, and goroutine management for building robust network services.
package gtudpd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMaxConcurrentResponses is the default limit for concurrent response goroutines
	DefaultMaxConcurrentResponses = 500
	// DefaultMaxRequestSize is the default maximum UDP packet size to accept
	DefaultMaxRequestSize = 64
	// DefaultMaxRequestsPerIP is the default rate limit per IP
	DefaultMaxRequestsPerIP = 100
	// DefaultRateLimitWindow is the default time window for rate limiting
	DefaultRateLimitWindow = 1 * time.Second
	// DefaultResponseTimeout is the default timeout for response operations
	DefaultResponseTimeout = 1 * time.Second
	// DefaultReadTimeout is the default UDP read timeout to prevent blocking
	DefaultReadTimeout = 10 * time.Millisecond
)

// RequestHandler defines the interface for handling UDP requests
type RequestHandler func(conn *net.UDPConn, n int, remoteaddr *net.UDPAddr, buf []byte)

// RequestValidator defines the interface for validating requests
type RequestValidator func(n int, buf []byte, remoteIP net.IP) bool

// Config holds server configuration including port and access control
type Config struct {
	DefaultPort            string
	ConfigDir              string
	MaxConcurrentResponses int
	MaxRequestSize         int
	MaxRequestsPerIP       int
	RateLimitWindow        time.Duration
	ResponseTimeout        time.Duration
	ReadTimeout            time.Duration
}

// Server represents a UDP server with rate limiting and security features
type Server struct {
	conn              *net.UDPConn
	handler           RequestHandler
	validator         RequestValidator
	responseSemaphore chan struct{}
	rateLimitMutex    sync.RWMutex
	rateLimitMap      map[string]*ipRateLimit
	ctx               context.Context
	cancel            context.CancelFunc
	config            *Config
	ipStringCache     sync.Map
}

type ipRateLimit struct {
	requests int
	window   time.Time
}

// validatePortFilePath checks if the port file path is safe to read
func validatePortFilePath(portFile string, configDir string) bool {
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		return false
	}
	absPortFile, err := filepath.Abs(portFile)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absPortFile, absConfigDir+string(filepath.Separator))
}

// extractPortNumber extracts port number from string, handling colon prefix
func extractPortNumber(portStr string) (string, bool) {
	if len(portStr) > 0 && portStr[0] == ':' {
		if len(portStr) == 1 {
			return "", false // Just a colon with no port number
		}
		return portStr[1:], true
	}
	return portStr, false
}

// validatePortNumber checks if port number is in valid range
func validatePortNumber(portNum string) bool {
	port, err := strconv.Atoi(portNum)
	return err == nil && port > 0 && port <= 65535
}

// parsePortString extracts and validates port number from string
func parsePortString(portStr string) (string, bool) {
	if portStr == "" {
		return "", false
	}

	portNum, hasColon := extractPortNumber(portStr)
	if !validatePortNumber(portNum) {
		return "", false
	}

	if hasColon {
		return portStr, true
	}
	return ":" + portStr, true
}

// GetPort reads the port configuration from the config directory.
// Returns the port in ":port" format, falling back to defaultPort if not configured.
func (config *Config) GetPort() string {
	if config.ConfigDir == "" {
		return config.DefaultPort
	}

	portFile := filepath.Join(config.ConfigDir, "port")
	if !validatePortFilePath(portFile, config.ConfigDir) {
		return config.DefaultPort
	}

	data, err := os.ReadFile(portFile)
	if err != nil {
		return config.DefaultPort
	}

	portStr := strings.TrimSpace(string(data))
	if result, valid := parsePortString(portStr); valid {
		return result
	}
	return config.DefaultPort
}

// isValidIPChar checks if a character is valid for IP addresses
func isValidIPChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') ||
		r == '.' || r == ':'
}

// isValidNetworkName validates that a network name only contains valid IP characters.
func isValidNetworkName(name string) bool {
	if name == "" {
		return false
	}

	for _, r := range name {
		if !isValidIPChar(r) {
			return false
		}
	}
	return true
}

// checkNetworkFile checks if a network file exists in the config directory.
func (config *Config) checkNetworkFile(network string) bool {
	// Validate network name contains only valid IP characters
	if !isValidNetworkName(network) {
		return false
	}

	networkFile := filepath.Join(config.ConfigDir, network)
	// Validate that the resolved path is within the config directory
	absConfigDir, err := filepath.Abs(config.ConfigDir)
	if err != nil {
		return false
	}
	absNetworkFile, err := filepath.Abs(networkFile)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(absNetworkFile, absConfigDir+string(filepath.Separator)) {
		return false
	}

	_, err = os.Stat(networkFile)
	return err == nil
}

// checkIPv4Networks checks IPv4 network patterns for clientok.
func (config *Config) checkIPv4Networks(ip4 net.IP) bool {
	// Check /8 network
	network8 := strconv.Itoa(int(ip4[0]))
	if config.checkNetworkFile(network8) {
		return true
	}

	// Check /16 network
	network16 := strconv.Itoa(int(ip4[0])) + "." + strconv.Itoa(int(ip4[1]))
	if config.checkNetworkFile(network16) {
		return true
	}

	// Check /24 network
	network24 := strconv.Itoa(int(ip4[0])) + "." + strconv.Itoa(int(ip4[1])) + "." + strconv.Itoa(int(ip4[2]))
	return config.checkNetworkFile(network24)
}

// ClientOK checks if the given IP address is allowed to use the server.
// It follows DJB's clientok pattern by checking for the existence of files
// in the config directory matching the IP address.
// Special case: a file named "0" allows all clients.
func (config *Config) ClientOK(ip net.IP) bool {
	if config.ConfigDir == "" {
		return true // If no config directory specified, allow all
	}

	// Check for "0" file which allows all clients
	if config.checkNetworkFile("0") {
		return true
	}

	// For IPv4, check for network matches starting with broadest (/8)
	if ip4 := ip.To4(); ip4 != nil {
		if config.checkIPv4Networks(ip4) {
			return true
		}
	}

	// Check for exact IP match last
	return config.checkNetworkFile(ip.String())
}

// setConfigDefaults initializes Config fields with default values if they are zero
func setConfigDefaults(config *Config) {
	if config.MaxConcurrentResponses <= 0 {
		config.MaxConcurrentResponses = DefaultMaxConcurrentResponses
	}
	if config.MaxRequestSize <= 0 {
		config.MaxRequestSize = DefaultMaxRequestSize
	}
	if config.MaxRequestsPerIP <= 0 {
		config.MaxRequestsPerIP = DefaultMaxRequestsPerIP
	}
	if config.RateLimitWindow <= 0 {
		config.RateLimitWindow = DefaultRateLimitWindow
	}
	if config.ResponseTimeout <= 0 {
		config.ResponseTimeout = DefaultResponseTimeout
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = DefaultReadTimeout
	}
}

// NewServer creates a new UDP server with configuration
func NewServer(config *Config, handler RequestHandler, validator RequestValidator) (*Server, error) {
	setConfigDefaults(config)
	port := config.GetPort()
	servAddr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		return nil, err
	}

	servConn, err := net.ListenUDP("udp", servAddr)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return nil, fmt.Errorf(
				"permission denied binding to port %s - try running as root or use a port >= 1024", port)
		}
		return nil, err
	}

	// Set read timeout to prevent blocking on slow connections
	err = servConn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
	if err != nil {
		_ = servConn.Close()
		return nil, fmt.Errorf("failed to set read deadline: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		conn:              servConn,
		handler:           handler,
		validator:         validator,
		responseSemaphore: make(chan struct{}, config.MaxConcurrentResponses),
		rateLimitMap:      make(map[string]*ipRateLimit),
		ctx:               ctx,
		cancel:            cancel,
		config:            config,
	}

	// Start cleanup routine
	go server.cleanupRateLimit()

	return server, nil
}

// Start begins processing UDP requests
func (s *Server) Start() {
	buf := make([]byte, s.config.MaxRequestSize)
	s.handleClientRequests(buf)
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.cancel()
	return s.conn.Close()
}

// Addr returns the server's listening address
func (s *Server) Addr() net.Addr {
	return s.conn.LocalAddr()
}

// checkRateLimit checks if an IP is within rate limits
func (s *Server) checkRateLimit(ip string) bool {
	now := time.Now()

	s.rateLimitMutex.Lock()
	defer s.rateLimitMutex.Unlock()

	limit, exists := s.rateLimitMap[ip]
	if !exists {
		s.rateLimitMap[ip] = &ipRateLimit{requests: 1, window: now}
		return true
	}

	// Reset window if expired
	if now.Sub(limit.window) > s.config.RateLimitWindow {
		limit.requests = 1
		limit.window = now
		return true
	}

	// Check if within limits
	if limit.requests >= s.config.MaxRequestsPerIP {
		return false
	}

	limit.requests++
	return true
}

// getIPString returns cached IP string to avoid repeated allocations
func (s *Server) getIPString(ip net.IP) string {
	if cached, ok := s.ipStringCache.Load(ip.String()); ok {
		if ipStr, isString := cached.(string); isString {
			return ipStr
		}
	}
	ipStr := ip.String()
	s.ipStringCache.Store(ipStr, ipStr)
	return ipStr
}

// cleanupExpiredEntries removes old rate limit entries
func (s *Server) cleanupExpiredEntries() {
	s.rateLimitMutex.Lock()
	defer s.rateLimitMutex.Unlock()

	now := time.Now()
	for ip, limit := range s.rateLimitMap {
		if now.Sub(limit.window) > s.config.RateLimitWindow*2 {
			delete(s.rateLimitMap, ip)
		}
	}
}

// cleanupRateLimit periodically cleans up old rate limit entries
func (s *Server) cleanupRateLimit() {
	ticker := time.NewTicker(s.config.RateLimitWindow * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredEntries()
		case <-s.ctx.Done():
			return
		}
	}
}

// processRequest handles validation and response for a single request
func (s *Server) processRequest(n int, remoteaddr *net.UDPAddr, buf []byte) {
	// Check rate limit first (cheapest check)
	ipStr := s.getIPString(remoteaddr.IP)
	if !s.checkRateLimit(ipStr) {
		return // Drop request due to rate limiting
	}

	if !s.validator(n, buf, remoteaddr.IP) {
		return
	}

	select {
	case s.responseSemaphore <- struct{}{}:
		// Successfully acquired semaphore slot
		go s.handleResponse(n, remoteaddr, buf)
	default:
		// No available slots, drop the request
		// This prevents resource exhaustion
	}
}

// handleResponse processes the response with timeout protection
func (s *Server) handleResponse(n int, remoteaddr *net.UDPAddr, buf []byte) {
	defer func() { <-s.responseSemaphore }() // Release semaphore when done

	// Create a timeout context for the response operation
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ResponseTimeout)
	defer cancel()

	// Channel to signal completion
	done := make(chan error, 1)

	go func() {
		s.handler(s.conn, n, remoteaddr, buf)
		done <- nil
	}()

	select {
	case <-done:
		// Response completed successfully
	case <-ctx.Done():
		// Response timed out
	}
}

// handleReadError processes UDP read errors
func (*Server) handleReadError(err error) bool {
	// Handle timeout errors silently to avoid log spam
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return false // Continue processing
	}
	return false // Continue processing
}

// handleClientRequests processes incoming client requests
func (s *Server) handleClientRequests(buf []byte) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Reset read deadline for each iteration
		if err := s.conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
			continue
		}

		n, remoteaddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			s.handleReadError(err)
			continue
		}

		s.processRequest(n, remoteaddr, buf)
	}
}
