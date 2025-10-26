package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true }, // Allow all origins
	HandshakeTimeout: 100000,
	ReadBufferSize:   0, // buffer allcated by HTTP server used here
	WriteBufferSize:  0,
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

// make this a thread-safe structure in production, or add mutex
var connectedClients = []string{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	// custom defer to track client disconnection
	defer func() {
		logger.Info("Client disconnected", "address", r.RemoteAddr)
		//log.Printf("Client disconnected: %s", r.RemoteAddr)
		connectedClients = connectedClients[:len(connectedClients)-1]
		conn.Close()
	}()

	// Track connected client
	currentClient := r.RemoteAddr
	connectedClients = append(connectedClients, currentClient)
	log.Printf("Client connected: %s", currentClient)

	for {
		// Read message from client (blocks until message received)
		log.Printf("Total active clients: %d", len(connectedClients))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received: %s", msg)

		// Echo message back to client
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
