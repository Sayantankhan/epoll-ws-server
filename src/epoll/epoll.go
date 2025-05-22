package epoll

import (
	"fmt"
	"sync"

	"golang.org/x/sys/unix"
)

type ConnHandler interface {
	Read(fd int)
	FlushBuffer()
	Close()
}

type Epoll struct {
	epfd  int
	conns map[int]ConnHandler
	mu    sync.RWMutex
}

func NewEpoll() (*Epoll, error) {
	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, fmt.Errorf("epoll create: %w", err)
	}
	return &Epoll{
		epfd:  epfd,
		conns: make(map[int]ConnHandler),
	}, nil
}

func (e *Epoll) Add(fd int, handler ConnHandler) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.conns[fd] = handler
	event := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
	return unix.EpollCtl(e.epfd, unix.EPOLL_CTL_ADD, fd, event)
}

func (e *Epoll) Wait(events []int) ([]int, error) {
	epEvents := make([]unix.EpollEvent, len(events))

	for {
		n, err := unix.EpollWait(e.epfd, epEvents, -1)
		if err != nil {
			return nil, fmt.Errorf("epoll wait: %w", err)
		}

		var fds []int
		for i := 0; i < n; i++ {
			fds = append(fds, int(epEvents[i].Fd))
		}
		return fds, nil
	}
}

func (e *Epoll) Read(fd int) {
	e.mu.RLock()
	handler, ok := e.conns[fd]
	e.mu.RUnlock()

	if ok {
		handler.Read(fd)
	}
}

func (e *Epoll) FlushAll() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, conn := range e.conns {
		conn.FlushBuffer()
	}
}

func (e *Epoll) Remove(fd int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = unix.EpollCtl(e.epfd, unix.EPOLL_CTL_DEL, fd, nil)

	if conn, ok := e.conns[fd]; ok {
		conn.Close() // Interface call, not tied to WSConn
		delete(e.conns, fd)
	}
}
