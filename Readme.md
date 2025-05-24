epoll-ws-server
---------------------
A high-performance WebSocket server built in Go, leveraging Linux's epoll for efficient handling of multiple simultaneous connections. Mimicking game server communication for scalable and responsive multiplayer interactions.

🚀 Features
----------
- Efficient I/O multiplexing with epoll
- Low-level WebSocket frame parsing
- SHA-256 hashing of client messages

- Message broadcasting to all connected clients

- Optimized for real-time game communication scenarios

📁 Project Structure
----------------
```
epoll-ws-server/
├── go.mod
├── go.sum
└── src/
    ├── epoll/
    │   ├── epoll.go        # Epoll instance management
    │   └── wsconn.go       # WebSocket connection handling
    └── server/
        └── server.go       # HTTP server setup and main loop
```

🛠️ Installation
----------------
1. Clone the repository:
```bash
git clone https://github.com/Sayantankhan/epoll-ws-server.git
```

2. Build
```bash
cd epoll-ws-server
go build -o epoll-ws-server ./src/server
```

3. Run the server:
```bash
go build -o epoll-ws-server ./src/server
```

🚦 Usage
---------
Run the server:
```bash
./epoll-ws-server
```
The server will start and listen on 0.0.0.0:9009.

Connect via WebSocket:
------------

Clients can connect to the server using the WebSocket protocol at ws://localhost:9009/ws.

🧪 Testing
-----------
You can test the WebSocket server using tools like websocat or browser-based WebSocket clients.

