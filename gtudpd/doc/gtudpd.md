# gtudpd - Secure UDP Server Package

The `gtudpd` package provides a secure, high-performance UDP server
implementation with built-in rate limiting, access control, and worker pool
architecture for building robust network services.

## Features

- **Rate Limiting**: Per-IP request limiting with configurable windows
- **Access Control**: DJB-style clientok pattern for IP-based access
  control
- **Worker Pool**: Prevents resource exhaustion with configurable concurrent
  response limits
- **Security**: Path traversal protection and input validation
- **Performance**: IP string caching and optimized goroutine management
- **Timeout Protection**: Configurable timeouts for read and response operations

## Architecture

The server uses a worker pool architecture to handle requests efficiently:

1. Main goroutine listens for UDP packets
2. Incoming requests are validated (rate limit, access control,
   custom validation)
3. Valid requests are queued to a worker pool channel
4. Worker goroutines process requests from the pool
5. Rate limit cleanup runs periodically to free memory

## Configuration

### Config Structure

```go
type Config struct {
    DefaultPort            string        // Default port (e.g., ":4014")
    ConfigDir              string        // Directory for configuration files
    MaxConcurrentResponses int           // Max concurrent worker goroutines
    MaxRequestSize         int           // Max UDP packet size to accept
    MaxRequestsPerIP       int           // Rate limit per IP
    RateLimitWindow        time.Duration // Rate limit time window
    ResponseTimeout        time.Duration // Timeout for response operations
    ReadTimeout            time.Duration // UDP read timeout
}
```

### Default Values

| Setting | Default | Description |
|---------|---------|-------------|
| MaxConcurrentResponses | 500 | Maximum concurrent response goroutines |
| MaxRequestSize | 64 bytes | Maximum UDP packet size |
| MaxRequestsPerIP | 100 | Requests per IP per window |
| RateLimitWindow | 1 second | Rate limiting time window |
| ResponseTimeout | 1 second | Response operation timeout |
| ReadTimeout | 10ms | UDP read timeout |

## Access Control (ClientOK)

The server implements DJB's clientok pattern using files in the config
directory:

- **File `0`**: Allows all clients (overrides all other rules)
- **Exact IP**: File named with exact IP (e.g., `127.0.0.1`, `::1`)
- **Network prefixes**:
  - `/8` network: file `192`
  - `/16` network: file `192.168`
  - `/24` network: file `192.168.1`

The server checks in order: file `0` → network matches (broadest first)
→ exact IP match.

## Port Configuration

Ports can be configured in two ways:

1. **Static**: Set `DefaultPort` in Config (e.g., `:4014`)
2. **Dynamic**: Create `port` file in `ConfigDir` containing port
   number

Port file format:

- Plain number: `8080` → becomes `:8080`
- With colon: `:9000` → used as-is
- Whitespace is trimmed
- Must be in range 1-65535

## Handler and Validator Functions

### RequestHandler

```go
type RequestHandler func(conn *net.UDPConn, n int,
                         remoteaddr *net.UDPAddr, buf []byte)
```

The handler function processes valid requests. It receives:

- `conn`: UDP connection for sending responses
- `n`: Number of bytes received
- `remoteaddr`: Client's UDP address
- `buf`: Request data buffer

### RequestValidator

```go
type RequestValidator func(n int, buf []byte, remoteIP net.IP) bool
```

The validator function performs custom request validation. It receives:

- `n`: Number of bytes received
- `buf`: Request data buffer
- `remoteIP`: Client's IP address

Return `true` to accept the request, `false` to reject it.

## API Reference

### NewServer

```go
func NewServer(config *Config, handler RequestHandler,
               validator RequestValidator) (*Server, error)
```

Creates a new UDP server instance. The server will bind to the
configured port and initialize the worker pool.

### Server Methods

```go
func (s *Server) Start()                 // Begin processing requests (blocking)
func (s *Server) Stop() error            // Gracefully shutdown server
func (s *Server) Addr() net.Addr         // Get listening address
```

### Config Methods

```go
func (config *Config) GetPort() string          // Get effective port
func (config *Config) ClientOK(ip net.IP) bool  // Check IP access
```

## Error Handling

The server handles various error conditions:

- **Permission denied**: Provides helpful error message for privileged ports
- **Rate limiting**: Silently drops requests exceeding limits
- **Timeouts**: Handles UDP read timeouts gracefully
- **Validation failures**: Drops invalid requests without response
- **Worker pool full**: Drops requests when workers are busy

## Memory Management

- **Rate limit cleanup**: Automatic cleanup of expired rate limit entries
- **IP string caching**: Reduces allocation overhead for repeated IPs
- **Buffer management**: Copies request buffers to prevent race conditions
- **Graceful shutdown**: Proper cleanup of resources on server stop
