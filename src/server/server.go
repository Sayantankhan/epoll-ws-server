package server

import (
	"epoll-server/src/epoll"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	HOST          = "0.0.0.0"
	PORT          = 9009
	TICK_INTERVAL = 2000 * time.Millisecond
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
		for _, fd := range fds {
			go ep.Read(fd)
		}

		// ep.FlushAll()
	}
}
