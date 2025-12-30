package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listenTCPArgs struct {
	Port Port `json:"port" jsonschema:"tcp port to listen"`
}

type listenTCPResult struct {
	Port      Port   `json:"port" jsonschema:"tcp port"`
	PublicURL string `json:"public_url,omitempty" jsonschema:"public tunnel URL"`
}

type listenHTTPArgs struct {
	Port Port `json:"port" jsonschema:"http port to listen"`
}

type listenHTTPResult struct {
	Port      Port   `json:"port" jsonschema:"http port"`
	PublicURL string `json:"public_url,omitempty" jsonschema:"public tunnel URL"`
}

type closeTCPArgs struct {
	Port Port `json:"port" jsonschema:"tcp port to stop"`
}

type closeTCPResult struct {
	Port Port `json:"port" jsonschema:"tcp port"`
}

type closeHTTPArgs struct {
	Port Port `json:"port" jsonschema:"http port to stop"`
}

type closeHTTPResult struct {
	Port Port `json:"port" jsonschema:"http port"`
}

type statusArgs struct{}

type statusResult struct {
	Listeners []map[string]any  `json:"listeners" jsonschema:"active listeners and connections"`
	Errors    []BackgroundError `json:"errors" jsonschema:"background errors"`
}

type sendTCPArgs struct {
	Port     Port   `json:"port" jsonschema:"tcp port"`
	Data     string `json:"data" jsonschema:"payload"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 or base64"`
}

type sendTCPResult struct {
	Port  Port `json:"port" jsonschema:"tcp port"`
	Bytes int  `json:"bytes" jsonschema:"bytes sent"`
}

type readTCPArgs struct {
	Port     Port   `json:"port" jsonschema:"tcp port"`
	Offset   int64  `json:"offset,omitempty" jsonschema:"read offset"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max bytes to read"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 or base64"`
}

type readTCPResult struct {
	Port      Port   `json:"port" jsonschema:"tcp port"`
	Offset    int64  `json:"offset" jsonschema:"requested offset"`
	Next      int64  `json:"next" jsonschema:"next offset"`
	Total     int64  `json:"total" jsonschema:"total bytes available"`
	Truncated bool   `json:"truncated" jsonschema:"true when offset was behind buffer"`
	Encoding  string `json:"encoding" jsonschema:"utf8 or base64"`
	Data      string `json:"data" jsonschema:"payload"`
}

type readHTTPArgs struct {
	Port     Port   `json:"port" jsonschema:"http port"`
	Offset   int64  `json:"offset,omitempty" jsonschema:"read offset"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max bytes to read"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 or base64"`
}

type readHTTPResult struct {
	Port      Port   `json:"port" jsonschema:"http port"`
	Offset    int64  `json:"offset" jsonschema:"requested offset"`
	Next      int64  `json:"next" jsonschema:"next offset"`
	Total     int64  `json:"total" jsonschema:"total bytes available"`
	Truncated bool   `json:"truncated" jsonschema:"true when offset was behind buffer"`
	Encoding  string `json:"encoding" jsonschema:"utf8 or base64"`
	Data      string `json:"data" jsonschema:"payload"`
}

func decodeData(data string, encoding string) ([]byte, error) {
	switch encoding {
	case "", "utf8":
		return []byte(data), nil
	case "base64":
		return base64.StdEncoding.DecodeString(data)
	default:
		return nil, fmt.Errorf("unsupported encoding %q", encoding)
	}
}

func encodeData(data []byte, encoding string) (string, error) {
	switch encoding {
	case "", "utf8":
		return string(data), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(data), nil
	default:
		return "", fmt.Errorf("unsupported encoding %q", encoding)
	}
}

func newMCPServer(mgr *ListenerManager) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "oob-probe-mcp",
		Title:   "OOB probe MCP server",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "status", Description: "Show listeners, connections, and async errors."},
		func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
			result := statusResult{
				Listeners: mgr.GetStatus(),
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "listen_tcp", Description: "Start listening on a TCP port."},
		func(ctx context.Context, req *mcp.CallToolRequest, args listenTCPArgs) (*mcp.CallToolResult, any, error) {
			l, err := mgr.ListenTCP(args.Port)
			if err != nil {
				return nil, nil, err
			}
			result := listenTCPResult{
				Port:      l.backendPort,
				PublicURL: l.tunnel.URL(),
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "listen_http", Description: "Start listening on an HTTP port."},
		func(ctx context.Context, req *mcp.CallToolRequest, args listenHTTPArgs) (*mcp.CallToolResult, any, error) {
			l, err := mgr.ListenHTTP(args.Port)
			if err != nil {
				return nil, nil, err
			}
			result := listenHTTPResult{
				Port:      l.backendPort,
				PublicURL: l.tunnel.URL(),
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "close_tcp", Description: "Stop listening on a TCP port and close connections."},
		func(ctx context.Context, req *mcp.CallToolRequest, args closeTCPArgs) (*mcp.CallToolResult, any, error) {
			if err := mgr.CloseTCP(args.Port); err != nil {
				return nil, nil, err
			}
			return nil, closeTCPResult{Port: args.Port}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "close_http", Description: "Stop listening on an HTTP port."},
		func(ctx context.Context, req *mcp.CallToolRequest, args closeHTTPArgs) (*mcp.CallToolResult, any, error) {
			if err := mgr.CloseHTTP(args.Port); err != nil {
				return nil, nil, err
			}
			return nil, closeHTTPResult{Port: args.Port}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "send_tcp", Description: "Send data to a TCP connection."},
		func(ctx context.Context, req *mcp.CallToolRequest, args sendTCPArgs) (*mcp.CallToolResult, any, error) {
			data, err := decodeData(args.Data, args.Encoding)
			if err != nil {
				return nil, nil, err
			}

			l, ok := mgr.GetTCP(args.Port)
			if !ok {
				return nil, nil, fmt.Errorf("port %d not listening (tcp)", args.Port)
			}

			if err := l.Send(data); err != nil {
				return nil, nil, err
			}
			return nil, sendTCPResult{Port: args.Port, Bytes: len(data)}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "read_tcp", Description: "Read TCP log bytes for a port with an offset/limit for scrolling."},
		func(ctx context.Context, req *mcp.CallToolRequest, args readTCPArgs) (*mcp.CallToolResult, any, error) {
			l, ok := mgr.GetTCP(args.Port)
			if !ok {
				return nil, nil, fmt.Errorf("port %d not listening (tcp)", args.Port)
			}

			data, next, total, truncated := l.Read(args.Offset, args.Limit)
			encoded, err := encodeData(data, args.Encoding)
			if err != nil {
				return nil, nil, err
			}

			result := readTCPResult{
				Port:      args.Port,
				Offset:    args.Offset,
				Next:      next,
				Total:     total,
				Truncated: truncated,
				Encoding:  args.Encoding,
				Data:      encoded,
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "read_http", Description: "Read HTTP log bytes for a port with an offset/limit for scrolling."},
		func(ctx context.Context, req *mcp.CallToolRequest, args readHTTPArgs) (*mcp.CallToolResult, any, error) {
			l, ok := mgr.GetHTTP(args.Port)
			if !ok {
				return nil, nil, fmt.Errorf("port %d not listening (http)", args.Port)
			}

			data, next, total, truncated := l.Read(args.Offset, args.Limit)
			encoded, err := encodeData(data, args.Encoding)
			if err != nil {
				return nil, nil, err
			}

			result := readHTTPResult{
				Port:      args.Port,
				Offset:    args.Offset,
				Next:      next,
				Total:     total,
				Truncated: truncated,
				Encoding:  args.Encoding,
				Data:      encoded,
			}
			return nil, result, nil
		})

	return server
}
