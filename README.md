# GO-RTCS

A **real-time chat server** built in **Go**, where multiple users can connect and exchange messages in a single chat room. The server demonstrates **Go’s concurrency features** using goroutines and channels, and supports **real-time message broadcasting** via WebSockets.

---

## Features (MVP)

- **Real-Time Messaging**  
  Users can send and receive messages instantly in a single chat room. No authentication required; anyone can connect.

- **WebSocket Server**  
  Single `/ws` endpoint for connecting clients. Handles multiple concurrent connections efficiently using Go goroutines.

- **In-Memory User Management**  
  Tracks connected users in memory and broadcasts messages to all active users.

- **Minimal Client**  
  Lightweight HTML + JS page for testing, or CLI/tools like `wscat` can connect to the WebSocket endpoint.

- **Optional Persistence (Post-MVP)**  
  - Save chat history in a database (SQLite/PostgreSQL).  
  - API endpoints to fetch past messages (`GET /messages`) or send via REST (`POST /messages`).

---

## Tech Stack

- **Backend:** Go (`net/http`, `gorilla/websocket`)  
- **Frontend (optional):** Minimal HTML + JS  
- **Data Storage:** In-memory (optional DB for persistence)  
- **Deployment:** Localhost / Docker  

---

## Architecture Overview

1. Client connects to the WebSocket server.  
2. Server creates a goroutine per client to handle incoming messages.  
3. Messages are sent through a channel to the broadcast routine.  
4. Broadcast routine forwards messages to all connected clients in real-time.  
5. Optional persistence layer stores messages in DB for future retrieval.  

---

## Future Enhancements / Twists

- Multiple rooms / channels  
- User authentication (JWT)  
- Ephemeral messages (disappear after X seconds)  
- Chat analytics (active users, messages per minute)  
- Bot for moderation or automatic replies  
- Dockerized deployment / online demo  

---