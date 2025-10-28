package main

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true }, // Allow all origins
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   512, // buffer allocated by HTTP server used here
	WriteBufferSize:  512,
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			if t, ok := a.Value.Any().(time.Time); ok {
				return slog.Attr{
					Key:   slog.TimeKey,
					Value: slog.StringValue(t.Format("15:04:05")), //Mon Jan 2 15:04:05 MST 2006
				} // Remove the time attribute
			}
		}
		return a
	},
}))

var (
	connectedClients = make(map[string]*websocket.Conn)
	clientsMutex     sync.RWMutex
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Upgrade error", "error", err)
		return
	}

	// custom defer to track client disconnection
	defer func() {
		logger.Info("Client disconnected", "address", r.RemoteAddr)

		clientsMutex.Lock()
		delete(connectedClients, r.RemoteAddr)
		count := len(connectedClients)
		clientsMutex.Unlock()

		logger.Info("Total active clients", "count", count)
		conn.Close()
	}()

	// Track connected client
	currentClient := r.RemoteAddr
	clientsMutex.Lock()
	connectedClients[currentClient] = conn
	count := len(connectedClients)
	clientsMutex.Unlock()

	logger.Info("Client connected", "address", currentClient, "total", count)

	for {
		// Read message from client (blocks until message received)
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Read error", "error", err)
			break
		}
		logger.Info("Received message", "message", string(msg))

		// Echo message back to client
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			logger.Error("Write error", "error", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	logger.Info("WebSocket server started on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("Server error", "error", err)
	}
}
