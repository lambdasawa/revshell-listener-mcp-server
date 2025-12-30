package main

import (
	"errors"
	"fmt"
	"golang.ngrok.com/ngrok"
	"net"
	"sync"
)

type Listener struct {
	id          string
	backendPort Port

	tunnel ngrok.Forwarder
	ln     net.Listener
	conn   net.Conn

	logStore *LogStore
	errors   []string
}

type LogStore struct {
	mu         sync.Mutex
	buf        []byte
	baseOffset int64
	maxSize    int
}

func (l *LogStore) Append(p []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(p) == 0 {
		return
	}
	if l.maxSize > 0 && len(p) > l.maxSize {
		p = p[len(p)-l.maxSize:]
	}
	if l.maxSize > 0 && len(l.buf)+len(p) > l.maxSize {
		over := len(l.buf) + len(p) - l.maxSize
		l.buf = l.buf[over:]
		l.baseOffset += int64(over)
	}
	l.buf = append(l.buf, p...)
}

func (l *LogStore) Read(offset int64, limit int) (data []byte, next int64, total int64, truncated bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	total = l.baseOffset + int64(len(l.buf))
	if offset < l.baseOffset {
		offset = l.baseOffset
		truncated = true
	}
	start := int(offset - l.baseOffset)
	if start < 0 || start > len(l.buf) {
		return nil, offset, total, truncated
	}
	if limit <= 0 {
		limit = len(l.buf) - start
	}
	end := start + limit
	if end > len(l.buf) {
		end = len(l.buf)
	}
	data = append([]byte(nil), l.buf[start:end]...)
	next = l.baseOffset + int64(end)
	return data, next, total, truncated
}

func (l *Listener) Send(data []byte) error {
	if l.conn == nil {
		return errors.New("no active connections")
	}

	_, err := l.conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	l.logStore.Append(data)

	return nil
}

func (l *Listener) Read(offset int64, limit int) (data []byte, next int64, total int64, truncated bool) {
	return l.logStore.Read(offset, limit)
}

func (l *Listener) Close() error {
	if l.ln != nil {
		if err := l.ln.Close(); err != nil {
			return err
		}
	}

	if l.conn != nil {
		if err := l.conn.Close(); err != nil {
			return err
		}
	}

	if l.tunnel != nil {
		if err := l.tunnel.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (l *Listener) acceptLoop() {
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			l.errors = append(l.errors, fmt.Errorf("accept failed port=%d: %v", l.backendPort, err).Error())
			_ = l.Close()
			return
		}
		l.conn = conn

		sendDesktopNotification(fmt.Sprintf("New connection on port %d", l.backendPort))

		l.readLoop()
	}
}

func (l *Listener) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := l.conn.Read(buf)
		if n > 0 {
			l.logStore.Append(buf[:n])
		}
		if err != nil {
			l.errors = append(l.errors, fmt.Errorf("read failed port=%d: %v", l.backendPort, err).Error())
			_ = l.Close()
			return
		}
	}
}
