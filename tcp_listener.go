package main

import (
	"errors"
	"fmt"
	"golang.ngrok.com/ngrok"
	"net"
)

type TCPListener struct {
	id          string
	backendPort Port

	tunnel ngrok.Forwarder
	ln     net.Listener
	conn   net.Conn

	logStore *LogStore
	errors   []string
}

func (l *TCPListener) Send(data []byte) error {
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

func (l *TCPListener) Read(offset int64, limit int) (data []byte, next int64, total int64, truncated bool) {
	return l.logStore.Read(offset, limit)
}

func (l *TCPListener) Close() error {
	if l.tunnel != nil {
		if err := l.tunnel.Close(); err != nil {
			return fmt.Errorf("failed to close ngrok tunnel: %v", err)
		}
	}

	if l.ln != nil {
		if err := l.ln.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %v", err)
		}
	}

	if l.conn != nil {
		if err := l.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %v", err)
		}
	}

	return nil
}

func (l *TCPListener) acceptLoop() {
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

func (l *TCPListener) readLoop() {
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
