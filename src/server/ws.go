package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"epoll-server/src/epoll"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/sys/unix"
)

type WSConn struct {
	fd     int
	id     string
	buffer []byte
	mu     sync.Mutex
	ep     *epoll.Epoll
}

var ErrIncompleteFrame = errors.New("incomplete websocket frame")

func (w *WSConn) Read(fd int) ([]string, error) {
	var messages []string
	for {
		tmp := make([]byte, 1024)
		n, err := unix.Read(fd, tmp)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				// No more data right now
				break
			}
			log.Printf("read error on fd %d: %v", fd, err)
			return nil, err
		}
		if n == 0 {
			w.ep.Remove(fd)
			log.Printf("connection closed on fd %d", fd)
			return nil, nil
		}
		w.buffer = append(w.buffer, tmp[:n]...)
	}

	for {
		msg, rest, err := w.parseFrame(w.buffer)
		if err != nil {
			if errors.Is(err, ErrIncompleteFrame) {
				// Need more data, wait for next read event
				break
			}
			log.Printf("frame parse error on fd %d: %v", fd, err)
			return nil, err
		}
		w.buffer = rest

		if msg != nil {
			// Full message received, handle it
			// fmt.Printf("Full WS message from fd %d: %s\n", fd, string(msg))
			// TODO: send to game update queue, buffer, or process here
			messages = append(messages, string(msg))
		} else {
			// No full message yet
			break
		}
	}

	return messages, nil
}

func (w *WSConn) FlushBuffer() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, msg := range w.buffer {
		fmt.Printf("Buffered message from fd %d: %s\n", w.fd, msg)
	}
	w.buffer = nil // Clear buffer after flushing
}

func (w *WSConn) Close() {
	_ = unix.Close(w.fd) // Close the file descriptor
	log.Printf("connection closed on fd %d", w.fd)
}

func HandleWebSocket(ep *epoll.Epoll) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // if you want to accept all origins
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("upgrade error: %v", err), http.StatusInternalServerError)
			return
		}

		// Once upgraded, never use http.Error or WriteHeader again
		raw, ok := conn.UnderlyingConn().(*net.TCPConn)
		if !ok {
			log.Printf("underlying conn is not *net.TCPConn")
			_ = conn.Close()
			return
		}

		file, err := raw.File()
		if err != nil {
			log.Printf("fd error: %v", err)
			_ = conn.Close()
			return
		}

		fd := int(file.Fd())
		if err := unix.SetNonblock(fd, true); err != nil {
			log.Printf("set nonblock error: %v", err)
			_ = conn.Close()
			return
		}

		ws := &WSConn{
			fd: fd,
			id: uuid.New().String(),
			ep: ep,
		}

		if err := ep.Add(fd, ws); err != nil {
			log.Printf("epoll add error: %v", err)
			_ = conn.Close()
			return
		}

		log.Printf("WebSocket connected: fd %d", fd)
	}
}

func (w *WSConn) parseFrame(data []byte) ([]byte, []byte, error) {
	if len(data) < 2 {
		return nil, data, ErrIncompleteFrame
	}

	fin := (data[0] & 0x80) != 0
	opcode := data[0] & 0x0F
	mask := (data[1] & 0x80) != 0
	payloadLen := int(data[1] & 0x7F)

	if opcode == 0x8 {
		// Close frame
		log.Println("Close frame received")
		return nil, nil, errors.New("connection closed by client")
	}
	if opcode == 0x9 {
		// Ping frame, optional: respond with Pong frame here
		log.Println("Ping frame received")
		return nil, data[2:], nil
	}
	if opcode == 0xA {
		// Pong frame
		log.Println("Pong frame received")
		return nil, data[2:], nil
	}

	offset := 2
	if payloadLen == 126 {
		if len(data) < 4 {
			return nil, data, ErrIncompleteFrame
		}
		payloadLen = int(binary.BigEndian.Uint16(data[2:4]))
		offset += 2
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return nil, data, ErrIncompleteFrame
		}
		payloadLen64 := binary.BigEndian.Uint64(data[2:10])
		if payloadLen64 > (1 << 31) {
			return nil, nil, fmt.Errorf("payload too large: %d", payloadLen64)
		}
		payloadLen = int(payloadLen64)
		offset += 8
	}

	var maskingKey []byte
	if mask {
		if len(data) < offset+4 {
			return nil, data, ErrIncompleteFrame
		}
		maskingKey = data[offset : offset+4]
		offset += 4
	}

	if len(data) < offset+payloadLen {
		return nil, data, ErrIncompleteFrame
	}

	payload := data[offset : offset+payloadLen]

	if mask {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskingKey[i%4]
		}
	}

	rest := data[offset+payloadLen:]

	if fin {
		return payload, rest, nil
	} else {
		// Fragmented frames are not handled in this simple POC
		return nil, data, fmt.Errorf("fragmented frames not supported")
	}
}
