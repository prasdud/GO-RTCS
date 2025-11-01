# GO-RTCS

A **real-time chat server** built in **Go**, where multiple users can connect and exchange messages in a single chat room. The server demonstrates **Go’s concurrency features** using goroutines and channels, and supports **real-time message broadcasting** via WebSockets.

---

## TODO
- Add ping / pong heartbeat
- set connection timeouts
- add error type checking


## Potential issues
- Broadcast Logic:
  The broadcast loop skips sending the message back to the sender, which is good. However, if a client connection fails during broadcast, you only log the error and break the loop. Consider removing dead connections from the map to avoid memory leaks.

  
## Features (MVP)

- **Real-Time Messaging**  
  Users can send and receive messages instantly in a single chat room. No authentication required; anyone can connect.

- **WebSocket Server**  
  Single `/ws` endpoint for connecting clients. Handles multiple concurrent connections efficiently using Go goroutines.

- **In-Memory User Management**  
  Tracks connected users in memory and broadcasts messages to all active users.

- **Minimal Client**  
  Lightweight HTML + JS page for testing, or CLI/tools like `wscat` can connect to the WebSocket endpoint.

---

## Tech Stack

- **Backend:** Go (`net/http`, `gorilla/websocket`)  
- **Frontend (optional):** Minimal HTML + JS  
- **Data Storage:** In-memory
- **Deployment:** Localhost, Docker for final product  

---

## Architecture Overview

1. Client connects to the WebSocket server (global server).  
2. Client enters username on connection, this username is used to identify the client.
3. Server creates a goroutine per client to handle incoming messages.  
4. Messages are sent through a channel to the broadcast routine.  
5. Broadcast routine forwards messages to all connected clients in real-time.  
6. Server has a TTL that kicks in after all clients disconnect.
---

## Future Enhancements / Twists

- Private rooms with user auth
- Ephemeral messages (disappear after X seconds)  
- Real time chat analytics (active users, messages per minute)  

---