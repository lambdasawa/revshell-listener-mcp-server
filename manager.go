package main

import (
	"fmt"
	"github.com/google/uuid"
	"net"
)

type Port uint

type ListenerManager struct {
	listeners map[Port]*Listener
}

type BackgroundError struct {
	Time    string `json:"time" jsonschema:"timestamp"`
	Message string `json:"message" jsonschema:"error message"`
}

func NewListenerManager() *ListenerManager {
	m := &ListenerManager{
		listeners: make(map[Port]*Listener),
	}
	return m
}

func (m *ListenerManager) GetStatus() []map[string]any {
	items := make([]map[string]any, 0, len(m.listeners))
	for _, l := range m.listeners {
		items = append(items, map[string]any{
			"id":         l.id,
			"port":       l.backendPort,
			"public_url": l.tunnel.URL(),
			"connected":  l.conn != nil,
			"errors":     l.errors,
		})
	}
	return items
}

func (m *ListenerManager) Listen(port Port) (*Listener, error) {
	if _, exists := m.listeners[port]; exists {
		return nil, fmt.Errorf("port %d already listening", port)
	}

	l := &Listener{
		id:          uuid.New().String(),
		backendPort: port,
		logStore:    new(LogStore),
	}
	m.listeners[port] = l

	if err := l.StartNgrokTunnel(); err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("failed to start ngrok tunnel: %v", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %v %v", port, err)
	}
	l.ln = ln

	go l.acceptLoop()

	return l, nil
}

func (m *ListenerManager) Close(port Port) error {
	l, exists := m.listeners[port]
	if !exists {
		return fmt.Errorf("port %d not listening", port)
	}
	delete(m.listeners, port)
	return l.Close()
}

func (m *ListenerManager) CloseAll() {
	for _, l := range m.listeners {
		_ = l.Close() // TODO: handle error
	}

	m.listeners = make(map[Port]*Listener)
}

func (m *ListenerManager) Get(port Port) (*Listener, bool) {
	l, ok := m.listeners[port]
	return l, ok
}
