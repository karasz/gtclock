# gtudpd Examples

This document provides practical examples of using the gtudpd package to
build UDP servers.

## Basic Echo Server

```go
package main

import (
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"

    udp "github.com/karasz/gtclock/gtudpd"
)

func main() {
    // Configuration
    config := &udp.Config{
        DefaultPort: ":8080",
        ConfigDir:   "./config", // Optional: for access control files
    }

    // Create server with echo handler and simple validator
    server, err := udp.NewServer(config, echoHandler, minLengthValidator)
    if err != nil {
        log.Fatal("Failed to create server:", err)
    }
    defer server.Stop()

    // Handle graceful shutdown
    go func() {
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, syscall.SIGTERM)
        <-c
        log.Println("Shutting down server...")
        server.Stop()
        os.Exit(0)
    }()

    log.Printf("Starting UDP server on %s", server.Addr())
    server.Start() // This blocks until server stops
}

// Echo handler - sends received data back to client
func echoHandler(conn *net.UDPConn, n int, remoteaddr *net.UDPAddr,
                 buf []byte) {
    response := make([]byte, n+6)
    copy(response, []byte("Echo: "))
    copy(response[6:], buf[:n])

    _, err := conn.WriteToUDP(response, remoteaddr)
    if err != nil {
        log.Printf("Failed to send response to %s: %v", remoteaddr, err)
    }
}

// Simple validator - require at least 4 bytes
func minLengthValidator(n int, buf []byte, remoteIP net.IP) bool {
    return n >= 4
}
```

## Time Server

```go
package main

import (
    "fmt"
    "log"
    "net"
    "time"

    udp "github.com/karasz/gtclock/gtudpd"
)

func main() {
    config := &udp.Config{
        DefaultPort:            ":1234",
        MaxConcurrentResponses: 100,
        MaxRequestSize:         32,
        MaxRequestsPerIP:       10,
        RateLimitWindow:        5 * time.Second,
    }

    server, err := udp.NewServer(config, timeHandler, timeValidator)
    if err != nil {
        log.Fatal(err)
    }
    defer server.Stop()

    log.Printf("Time server listening on %s", server.Addr())
    server.Start()
}

func timeHandler(conn *net.UDPConn, n int, remoteaddr *net.UDPAddr,
                 buf []byte) {
    request := string(buf[:n])
    var response []byte

    switch request {
    case "TIME":
        response = []byte(time.Now().Format(time.RFC3339))
    case "UNIX":
        response = []byte(fmt.Sprintf("%d", time.Now().Unix()))
    case "UTC":
        response = []byte(time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
    default:
        response = []byte("ERROR: Unknown command. Use TIME, UNIX, or UTC")
    }

    conn.WriteToUDP(response, remoteaddr)
}

func timeValidator(n int, buf []byte, remoteIP net.IP) bool {
    if n < 3 || n > 10 {
        return false
    }

    command := string(buf[:n])
    validCommands := []string{"TIME", "UNIX", "UTC"}

    for _, valid := range validCommands {
        if command == valid {
            return true
        }
    }
    return false
}
```

## Key-Value Store Server

```go
package main

import (
    "log"
    "net"
    "strings"
    "sync"

    udp "github.com/karasz/gtclock/gtudpd"
)

var (
    store = make(map[string]string)
    storeMutex sync.RWMutex
)

func main() {
    config := &udp.Config{
        DefaultPort:     ":5000",
        ConfigDir:       "./kvstore-config",
        MaxRequestSize:  256,
        MaxRequestsPerIP: 50,
    }

    server, err := udp.NewServer(config, kvHandler, kvValidator)
    if err != nil {
        log.Fatal(err)
    }
    defer server.Stop()

    log.Printf("Key-value store server on %s", server.Addr())
    server.Start()
}

func kvHandler(conn *net.UDPConn, n int, remoteaddr *net.UDPAddr, buf []byte) {
    request := string(buf[:n])
    parts := strings.Fields(request)

    var response string

    switch parts[0] {
    case "GET":
        if len(parts) == 2 {
            storeMutex.RLock()
            value, exists := store[parts[1]]
            storeMutex.RUnlock()

            if exists {
                response = "OK " + value
            } else {
                response = "NOT_FOUND"
            }
        } else {
            response = "ERROR Invalid GET syntax"
        }

    case "SET":
        if len(parts) >= 3 {
            key := parts[1]
            value := strings.Join(parts[2:], " ")

            storeMutex.Lock()
            store[key] = value
            storeMutex.Unlock()

            response = "OK"
        } else {
            response = "ERROR Invalid SET syntax"
        }

    case "DEL":
        if len(parts) == 2 {
            storeMutex.Lock()
            delete(store, parts[1])
            storeMutex.Unlock()

            response = "OK"
        } else {
            response = "ERROR Invalid DEL syntax"
        }

    default:
        response = "ERROR Unknown command"
    }

    conn.WriteToUDP([]byte(response), remoteaddr)
}

func kvValidator(n int, buf []byte, remoteIP net.IP) bool {
    if n < 3 {
        return false
    }

    request := string(buf[:n])
    parts := strings.Fields(request)

    if len(parts) == 0 {
        return false
    }

    validCommands := []string{"GET", "SET", "DEL"}
    for _, cmd := range validCommands {
        if parts[0] == cmd {
            return true
        }
    }

    return false
}
```

## Configuration Examples

### Access Control Setup

Create a config directory with access control files:

```bash
mkdir -p ./server-config

# Allow all clients from 192.168.x.x network
echo "" > ./server-config/192.168

# Allow specific IP
echo "" > ./server-config/10.0.0.100

# Allow all clients (overrides other rules)
echo "" > ./server-config/0
```

### Port Configuration

```bash
# Set port via file
echo "9090" > ./server-config/port

# Or with colon prefix
echo ":9090" > ./server-config/port
```

### High-Performance Configuration

```go
config := &udp.Config{
    DefaultPort:            ":8000",
    MaxConcurrentResponses: 1000,     // More workers for high load
    MaxRequestSize:         1024,     // Larger packets
    MaxRequestsPerIP:       1000,     // Higher rate limit
    RateLimitWindow:        1 * time.Second,
    ResponseTimeout:        500 * time.Millisecond,
    ReadTimeout:           5 * time.Millisecond,
}
```

### Restrictive Configuration

```go
config := &udp.Config{
    DefaultPort:            ":8000",
    ConfigDir:              "./secure-config", // Enable access control
    MaxConcurrentResponses: 50,       // Fewer workers
    MaxRequestSize:         64,       // Small packets only
    MaxRequestsPerIP:       10,       // Low rate limit
    RateLimitWindow:        10 * time.Second,
    ResponseTimeout:        2 * time.Second,
    ReadTimeout:           50 * time.Millisecond,
}
```

## Client Examples

### Simple UDP Client

```go
package main

import (
    "fmt"
    "net"
)

func main() {
    // Connect to server
    conn, err := net.Dial("udp", "localhost:8080")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Send request
    _, err = conn.Write([]byte("Hello Server"))
    if err != nil {
        panic(err)
    }

    // Read response
    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Server response: %s\n", string(buffer[:n]))
}
```

### Time Client

```bash
# Using netcat to test time server
echo -n "TIME" | nc -u localhost 1234
echo -n "UNIX" | nc -u localhost 1234
echo -n "UTC" | nc -u localhost 1234
```

### Key-Value Client

```bash
# Set a value
echo -n "SET name John Doe" | nc -u localhost 5000

# Get a value
echo -n "GET name" | nc -u localhost 5000

# Delete a value
echo -n "DEL name" | nc -u localhost 5000
```

## Testing Your Server

### Load Testing

```go
// Simple load test
func loadTest() {
    for i := 0; i < 1000; i++ {
        go func(id int) {
            conn, _ := net.Dial("udp", "localhost:8080")
            defer conn.Close()

            message := fmt.Sprintf("Test message %d", id)
            conn.Write([]byte(message))

            buffer := make([]byte, 1024)
            n, _ := conn.Read(buffer)
            fmt.Printf("Response %d: %s\n", id, string(buffer[:n]))
        }(i)
    }
}
```

### Rate Limit Testing

```bash
# Test rate limiting with rapid requests
for i in {1..200}; do
  echo -n "Test $i" | nc -u localhost 8080 &
done
wait
```

## Best Practices

1. **Always validate input** in both validator and handler functions
2. **Handle errors gracefully** in handlers - log but don't crash
3. **Use appropriate timeouts** based on your use case
4. **Configure rate limits** to prevent abuse
5. **Use access control** for production deployments
6. **Monitor server performance** and adjust worker pool size
7. **Implement graceful shutdown** for clean restarts
8. **Keep handlers fast** - offload heavy work to background goroutines
