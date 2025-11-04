package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true }, // Allow all origins, change this before deployment
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024, // buffer allocated by HTTP server used here
	WriteBufferSize:  1024,
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

type BroadcastMessage struct {
	//SenderUUID string
	Type     string `json:"Type"` // "message", "system", "prompt"
	UserName string `json:"UserName"`
	Message  string `json:"Message"`
	Time     string `json:"Time"`
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Upgrade error", "error", err)
		return
	}
	var hasUserName bool = false
	var userName string

	const maxMessageSize = 50
	conn.SetReadLimit(maxMessageSize)
	clientId := uuid.New().String()

	// custom defer to track client disconnection
	defer func() {
		//logger.Info("Client disconnected", "address", clientId)

		clientsMutex.Lock()
		delete(connectedClients, userName) // chaged from UUID to userName here
		count := len(connectedClients)
		clientsMutex.Unlock()

		logger.Info("Total active clients", "total", count)
		conn.Close()
	}()

	// Username entry loop - NO LOCK NEEDED HERE
	for !hasUserName {
		logger.Info("Waiting for username from client")

		// Send username prompt as JSON with special type
		promptMsg := BroadcastMessage{
			Type:     "prompt",
			UserName: "System",
			Message:  "Please enter username to enter chat room:",
			Time:     time.Now().Format("15:04:05"),
		}

		promptBytes, err := json.Marshal(promptMsg)
		if err != nil {
			logger.Error("Failed to marshal prompt", "error", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, promptBytes); err != nil {
			logger.Error("Failed to send username prompt", "error", err)
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Failed to read username", "error", err)
			return
		}

		userName = strings.TrimSpace(string(msg))

		if len(userName) > 0 {
			hasUserName = true
			logger.Info("User set username", "username", userName)

			// Send success message to clear prompt
			successMsg := BroadcastMessage{
				Type:     "success",
				UserName: "System",
				Message:  "Welcome, " + userName + "!",
				Time:     time.Now().Format("15:04:05"),
			}
			successBytes, _ := json.Marshal(successMsg)
			conn.WriteMessage(websocket.TextMessage, successBytes)
		} else {
			logger.Warn("Empty username received, prompting again")
		}
	}

	// NOW lock only when adding to the map
	clientsMutex.Lock()
	// Track connected client
	//clientId := r.RemoteAddr
	connectedClients[userName] = conn
	//connectedClients[clientId] = conn
	count := len(connectedClients)
	clientsMutex.Unlock()

	logger.Info("Client connected", "address", clientId, "username", userName, "total", count)

	for {
		// Read message from client (blocks until message received)
		_, msg, err := conn.ReadMessage()

		if err != nil {
			// Check for close errors and log gracefully
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Error("Client disconnected", "address", clientId, "username", userName, "error", err)
			} else {
				logger.Info("Read error", "error", err)
			}
			break
		}

		msgString := string(msg)

		if len(strings.ReplaceAll(msgString, " ", "")) == 0 {
			logger.Info("Empty message not allowed")
			continue
		}

		logger.Info("Received message", "message", strings.TrimSpace(msgString), "from", clientId)

		currentBroadcast := BroadcastMessage{
			//SenderUUID: clientId,
			Type:     "message",
			UserName: userName,
			Message:  msgString,
			Time:     time.Now().Format("15:04:05"),
		}

		// acquire lock to broadcast message to all clients
		// iterate through map of connected clients and send message
		clientsMutex.Lock()
		//currentConn := connectedClients[clientId]
		for _, clientConn := range connectedClients {
			msgBytes, err := json.Marshal(currentBroadcast)
			if err != nil {
				logger.Error("JSON marshal error", "error", err)
				continue
			}
			if err := clientConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				logger.Error("Broadcast error", "error", err)
				clientsMutex.Unlock()
				break
			}
		}
		clientsMutex.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	logger.Info("WebSocket server started on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("Server error", "error", err)
	}
}
