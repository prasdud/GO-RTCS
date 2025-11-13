# GO-RTCS V2 - Frankenstein Port

## Directory Structure

```
GO-RTCS/
│
├── cmd/
│   └── server/
│       └── main.go                    # Entry point - minimal, just wires everything
│
├── internal/                          # Private application code
│   ├── server/
│   │   ├── server.go                  # HTTP/WebSocket server setup
│   │   └── handlers.go                # HTTP handlers
│   │
│   ├── chat/
│   │   ├── hub.go                     # Central hub - manages all clients & broadcast
│   │   ├── client.go                  # Client struct & methods (read/write)
│   │   ├── message.go                 # Message types & validation
│   │
│   └── config/                         # global config manager
│       └── config.go
│
├── util/
│   └── logger/
│       └── logger.go                  # Custom logger setup
│
├── web/                               # Static files
│   └── static/
│       └── index.html                 # Client UI
│
│
├── go.mod
├── go.sum
├── Makefile                           # Build automation
├── Dockerfile
├── .gitignore
├── .env.example                       # Environment variables template
├── README.md
└── LICENSE
```

---

## Code Flow Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       cmd/server/main.go                     │
│  • Load config                                               │
│  • Initialize logger                                         │
│  • Create Hub                                                │
│  • Start Hub.Run() goroutine                                 │
│  • Setup HTTP server with handlers                           │
│  • Graceful shutdown                                         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                  internal/server/server.go                   │
│  • NewServer(hub, config)                                    │
│  • RegisterRoutes()                                          │
│  • Start() / Shutdown()                                      │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                internal/server/handlers.go                   │
│  • wsHandler(w, r) - WebSocket upgrade                       │
│  • serveHome(w, r) - Serve static HTML                       │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                    internal/chat/hub.go                      │
│                                                              │
│  Hub struct:                                                 │
│    • clients map[*Client]bool                                │
│    • broadcast chan *Message                                 │
│    • register chan *Client                                   │
│    • unregister chan *Client                                 │
│                                                              │
│  Methods:                                                    │
│    • Run() - Main event loop (select on channels)            │
│    • RegisterClient(client)                                  │
│    • UnregisterClient(client)                                │
│    • BroadcastMessage(msg)                                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                   internal/chat/client.go                    │
│                                                              │
│  Client struct:                                              │
│    • conn *websocket.Conn                                    │
│    • hub *Hub                                                │
│    • username string                                         │
│    • send chan []byte                                        │
│    • id string                                               │
│                                                              │
│  Methods:                                                    │
│    • readPump() - Read messages, send to hub                 │
│    • writePump() - Write messages from send channel          │
│    • disconnect()                                            │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                  internal/chat/message.go                    │
│                                                              │
│  Message struct:                                             │
│    • Type string (text, system, etc.)                        │
│    • Username string                                         │
│    • Content string                                          │
│    • Timestamp time.Time                                     │
│                                                              │
│  Methods:                                                    │
│    • Validate() error                                        │
│    • ToJSON() ([]byte, error)                                │
└─────────────────────────────────────────────────────────────┘
```

---

## Data Flow Diagram

```
CLIENT                    SERVER
  │                         │
  │   WebSocket Connect     │
  ├────────────────────────→│ wsHandler
  │                         │
  │                         ↓
  │                    [Create Client]
  │                         │
  │                         ↓
  │                    Hub.register ──→ [Hub.Run() goroutine]
  │                         │              │
  │   ← Username Prompt ────┤              │
  │                         │              │
  │   Send Username ───────→│              │
  │                         │              │
  │                         ↓              │
  │                    [Start readPump]    │
  │                    [Start writePump]   │
  │                         │              │
  │                         │              │
  │   Send Message ────────→│              │
  │                         │              │
  │                         ↓              │
  │                    readPump            │
  │                         │              │
  │                         ↓              │
  │              Hub.broadcast chan ──────→│
  │                                        │
  │                                        ↓
  │                              [Hub distributes to all
  │                               clients' send channels]
  │                                        │
  │                                        ↓
  │                         ┌──────────────┴──────────────┐
  │   ← Message ────────────┤ Client1.writePump           │
  │                         │ Client2.writePump           │
  │                         │ Client3.writePump           │
  │                         └─────────────────────────────┘
```

---

## Key Architectural Patterns

### 1. **Hub Pattern** (Centralized Message Distribution)
```
All clients → Hub (via channels) → All clients
```
- No direct client-to-client communication
- Hub manages all state in one goroutine (no mutex needed!)
- Channels handle concurrency

### 2. **Client Separation**
```
readPump goroutine  →  Hub  →  writePump goroutine
(reads from WS)              (writes to WS)
```
- One goroutine per direction per client
- Decoupled read/write operations

### 3. **Channel-Based Communication**
```
register chan      →  Add client
unregister chan    →  Remove client  
broadcast chan     →  Distribute message
client.send chan   →  Queue messages per client
```

### 4. **Dependency Injection**
```
main.go → creates Hub → passes to Server → passes to handlers
```

---

## File Responsibilities

| File | Responsibility | Size |
|------|---------------|------|
| `cmd/server/main.go` | Bootstrap only | ~50 lines |
| `internal/chat/hub.go` | Central orchestrator | ~150 lines |
| `internal/chat/client.go` | Per-client logic | ~200 lines |
| `internal/chat/message.go` | Data structures | ~50 lines |
| `internal/server/handlers.go` | HTTP/WS handlers | ~100 lines |
| `internal/config/config.go` | Config management | ~50 lines |
| `pkg/logger/logger.go` | Logging setup | ~30 lines |

---

## Go Best Practices Implemented

- ✅ `cmd/` for executables
- ✅ `internal/` for private code (cannot be imported by other projects)
- ✅ `pkg/` for reusable libraries
- ✅ Separation of concerns
- ✅ Testable components
- ✅ Channel-based concurrency (idiomatic Go)
- ✅ No shared state = no mutex hell
- ✅ Graceful shutdown
- ✅ Configuration via environment variables
- ✅ Structured logging

---

## Benefits of This Structure

1. **Testability** - Each package can be tested independently
2. **Maintainability** - Clear separation of concerns
3. **Scalability** - Easy to add features (rooms, auth, etc.)
4. **No Race Conditions** - Hub pattern eliminates mutex complexity
5. **Idiomatic Go** - Follows standard Go project layout
6. **Team Collaboration** - Clear boundaries for different developers

---

## Migration Path from V1

1. Create new directory structure
2. Move `server/main.go` → `cmd/server/main.go`
3. Extract Hub logic from main.go → `internal/chat/hub.go`
4. Extract Client logic → `internal/chat/client.go`
5. Extract Message types → `internal/chat/message.go`
6. Create server wrapper → `internal/server/server.go`
7. Move handlers → `internal/server/handlers.go`
8. Move `client/index.html` → `web/static/index.html`
9. Update imports
10. Test thoroughly

---

## Additional Improvements for V2

- [ ] Ping/pong heartbeat mechanism
- [ ] Connection timeouts
- [ ] Rate limiting per user
- [ ] Username uniqueness enforcement
- [ ] Graceful shutdown with SIGTERM/SIGINT
- [ ] Environment-based configuration
- [ ] Proper error types
- [ ] Unit tests for all packages
- [ ] Integration tests
- [ ] Metrics/monitoring endpoints
- [ ] Docker multi-stage build
- [ ] CI/CD pipeline
