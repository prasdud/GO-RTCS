# WebSocket Server Best Practices

## 1. DONE Client Identification
### Issue
```go
currentClient := r.RemoteAddr // Using IP:Port as key
```

### Problem
- Multiple clients behind NAT share same IP
- Clients reconnecting get same key → overwrites previous connection
- Port reuse can cause collisions

### Solution
```go
clientID := uuid.New().String() // Unique ID per connection
```

### Why
- Guarantees unique identification
- Survives reconnections
- Better for logging and tracking

---

## 2. Connection Timeouts
### Issue
```go
// No SetReadDeadline() or SetWriteDeadline()
```

### Problem
- Broken connections hang forever
- Dead clients consume resources
- No way to detect network issues

### Solution
```go
const (
    writeWait = 10 * time.Second
    pongWait  = 60 * time.Second
)

conn.SetReadDeadline(time.Now().Add(pongWait))
conn.SetWriteDeadline(time.Now().Add(writeWait))
```

### Why
- Detects stalled connections
- Frees resources automatically
- Prevents memory leaks

---

## 3. Ping/Pong Heartbeat
### Issue
```go
// No health check mechanism
```

### Problem
- Can't detect silently disconnected clients
- Zombie connections waste memory
- Client crashes go unnoticed

### Solution
```go
const pingPeriod = (pongWait * 9) / 10

ticker := time.NewTicker(pingPeriod)
go func() {
    for range ticker.C {
        conn.SetWriteDeadline(time.Now().Add(writeWait))
        if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            return
        }
    }
}()

conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(pongWait))
    return nil
})
```

### Why
- Actively probes connection health
- Client must respond to pong within `pongWait`
- Auto-cleanup of dead connections

---

## 4. DONE Message Size Limits
### Issue
```go
ReadBufferSize: 0  // Unlimited
```

### Problem
- Malicious clients can send huge messages
- Memory exhaustion attacks (DoS)
- Server crash from OOM

### Solution
```go
const maxMessageSize = 512 * 1024 // 512KB
conn.SetReadLimit(maxMessageSize)
```

### Why
- Prevents memory attacks
- Predictable resource usage
- Protects against mistakes

---

## 5. Graceful Shutdown
### Issue
```go
log.Fatal(http.ListenAndServe(":8080", nil)) // Immediate exit
```

### Problem
- Active connections forcefully closed
- Data loss during transmission
- No cleanup of resources

### Solution
```go
server := &http.Server{Addr: ":8080"}
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

go func() {
    server.ListenAndServe()
}()

<-stop
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
server.Shutdown(ctx)
```

### Why
- Allows in-flight requests to complete
- Graceful disconnect notifications
- Clean resource cleanup

---

## 6. Consistent Logging
### Issue
```go
log.Println("...")       // stdlib log
logger.Info("...")       // slog
log.Printf("...")        // mixed usage
```

### Problem
- Inconsistent log formats
- Hard to parse/filter
- Missing structured data

### Solution
```go
// Use only slog everywhere
logger.Info("Client connected", "id", clientID, "address", r.RemoteAddr)
logger.Error("Read error", "id", clientID, "error", err)
```

### Why
- Structured logging for easy parsing
- Consistent timestamps/format
- Better for log aggregation tools

---

## 7. Error Handling
### Issue
```go
if err != nil {
    log.Println("Read error:", err)
    break
}
```

### Problem
- Can't distinguish between normal/abnormal closes
- Expected closures logged as errors
- Noisy logs

### Solution
```go
if websocket.IsUnexpectedCloseError(err, 
    websocket.CloseGoingAway, 
    websocket.CloseAbnormalClosure) {
    logger.Error("Unexpected close", "error", err)
}
```

### Why
- Filters expected disconnections
- Cleaner logs
- Better error monitoring

---

## 8. Performance - Reduce Lock Contention
### Issue
```go
for {
    clientsMutex.RLock()
    count := len(connectedClients)
    clientsMutex.RUnlock()
    log.Printf("Total active clients: %d", count)  // Every loop iteration!
    
    _, msg, err := conn.ReadMessage()
}
```

### Problem
- Logs on **every message received**
- Unnecessary mutex operations
- Log spam
- Performance impact with many messages

### Solution
```go
// Only log count on connect/disconnect
defer func() {
    clientsMutex.Lock()
    delete(connectedClients, clientID)
    count := len(connectedClients)
    clientsMutex.Unlock()
    logger.Info("Active clients", "count", count)
}()
```

### Why
- Reduces lock contention
- Cleaner logs
- Better performance

---

## 9. HTTP Server Timeouts
### Issue
```go
http.ListenAndServe(":8080", nil) // No timeouts
```

### Problem
- Slowloris attacks
- Resource exhaustion
- Hanging connections

### Solution
```go
server := &http.Server{
    Addr:         ":8080",
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### Why
- Protects against slow clients
- Forces connection cleanup
- Production-ready defaults

---

## Priority Order for Implementation

1. **Critical (Security/Stability)**
   - Message size limits (#4)
   - Connection timeouts (#2)
   - Graceful shutdown (#5)

2. **High (Reliability)**
   - Ping/pong heartbeat (#3)
   - Client identification (#1)
   - Error handling (#7)

3. **Medium (Code Quality)**
   - Consistent logging (#6)
   - Performance optimization (#8)

4. **Low (Configuration)**
   - HTTP server timeouts (#9)
   - Upgrader config (#10)

---

## Testing Each Change

```bash
# Test connection timeout
# Connect client, don't send anything, wait 60s → should disconnect

# Test message size limit
# Send message > 512KB → should reject

# Test graceful shutdown
# Ctrl+C while client connected → clean disconnect message

# Test ping/pong
# Connect client, check logs for ping messages every 54s
```