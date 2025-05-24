package server

import (
	"crypto/sha256"
	"encoding/hex"
	"epoll-server/src/epoll"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	HOST          = "0.0.0.0"
	PORT          = 9009
	TICK_INTERVAL = 5000 * time.Millisecond
)

func Serve() {
	ep, err := epoll.NewEpoll()
	if err != nil {
		log.Fatalf("failed to create epoll: %v", err)
	}

	http.HandleFunc("/ws", HandleWebSocket(ep))

	go func() {
		addr := fmt.Sprintf("%s:%d", HOST, PORT)
		log.Printf("Starting server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	ticker := time.NewTicker(TICK_INTERVAL)
	defer ticker.Stop()

	events := make([]int, 100)
	for range ticker.C {
		fds, err := ep.Wait(events)
		if err != nil {
			log.Printf("epoll wait error: %v", err)
			continue
		}

		var hashes []string
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, fd := range fds {
			wg.Add(1)
			go func(fd int) {
				defer wg.Done()
				msgs, err := ep.Read(fd)
				if err != nil || len(msgs) == 0 {
					return
				}

				for _, msg := range msgs {
					fmt.Printf("Full WS message from fd %d: %s\n", fd, msg)
					hash := sha256.Sum256([]byte(msg))
					hashStr := hex.EncodeToString(hash[:])

					mu.Lock()
					hashes = append(hashes, hashStr)
					mu.Unlock()
				}
			}(fd)
		}

		wg.Wait()

		if len(hashes) == 0 {
			continue
		}

		final := strings.Join(hashes, " ")
		log.Printf("final output %s \v", final)
		//ep.Broadcast(final) // you define this

		// ep.FlushAll()
	}
}
