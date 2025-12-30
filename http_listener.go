package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"golang.ngrok.com/ngrok"
)

const maxHTTPBodyLogSize = 1024 * 1024

type HTTPListener struct {
	id          string
	backendPort Port

	tunnel ngrok.Forwarder
	ln     net.Listener
	server *http.Server

	logStore     *LogStore
	errors       []string
	requestCount int64
}

func (l *HTTPListener) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", l.backendPort))
	if err != nil {
		return fmt.Errorf("failed to listen on %v %v", l.backendPort, err)
	}
	l.ln = ln

	l.server = &http.Server{
		Handler: http.HandlerFunc(l.handle),
	}

	go func() {
		if err := l.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			l.errors = append(l.errors, fmt.Errorf("http serve failed port=%d: %v", l.backendPort, err).Error())
		}
	}()

	return nil
}

func (l *HTTPListener) Close() (returnErr error) {
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

	if l.server != nil {
		if err := l.server.Close(); err != nil {
			return fmt.Errorf("failed to close http server: %v", err)
		}
	}

	return nil
}

func (l *HTTPListener) Read(offset int64, limit int) (data []byte, next int64, total int64, truncated bool) {
	return l.logStore.Read(offset, limit)
}

func (l *HTTPListener) RequestCount() int64 {
	return atomic.LoadInt64(&l.requestCount)
}

func (l *HTTPListener) handle(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&l.requestCount, 1)
	sendDesktopNotification(fmt.Sprintf("New HTTP request on port %d", l.backendPort))

	body, truncated := readRequestBody(r)
	entry := formatHTTPRequestLog(r, body, truncated)
	l.logStore.Append([]byte(entry))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func readRequestBody(r *http.Request) ([]byte, bool) {
	if r.Body == nil {
		return nil, false
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, maxHTTPBodyLogSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return []byte(fmt.Sprintf("[body read error: %v]\n", err)), false
	}
	if len(data) > maxHTTPBodyLogSize {
		return data[:maxHTTPBodyLogSize], true
	}
	return data, false
}

func formatHTTPRequestLog(r *http.Request, body []byte, truncated bool) string {
	var b bytes.Buffer

	b.WriteString("----\n")
	b.WriteString(fmt.Sprintf("%s %s %s %s from %s\n",
		time.Now().Format(time.RFC3339),
		r.Method,
		r.URL.String(),
		r.Proto,
		r.RemoteAddr,
	))

	if r.Host != "" {
		b.WriteString("Host: ")
		b.WriteString(r.Host)
		b.WriteString("\n")
	}

	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(strings.Join(r.Header[k], ", "))
		b.WriteString("\n")
	}

	if len(body) > 0 {
		b.WriteString("\n")
		b.Write(body)
		if body[len(body)-1] != '\n' {
			b.WriteString("\n")
		}
	}
	if truncated {
		b.WriteString("[body truncated]\n")
	}
	b.WriteString("----\n")

	return b.String()
}
