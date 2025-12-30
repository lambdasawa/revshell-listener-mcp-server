package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listenArgs struct {
	Port Port `json:"port" jsonschema:"tcp port to listen"`
}

type listenResult struct {
	Port      Port   `json:"port" jsonschema:"tcp port"`
	PublicURL string `json:"public_url,omitempty" jsonschema:"public tunnel URL"`
}

type closeArgs struct {
	Port Port `json:"port" jsonschema:"tcp port to stop"`
}

type closeResult struct {
	Port Port `json:"port" jsonschema:"tcp port"`
}

type statusArgs struct{}

type statusResult struct {
	Listeners []map[string]any  `json:"listeners" jsonschema:"active listeners and connections"`
	Errors    []BackgroundError `json:"errors" jsonschema:"background errors"`
}

type sendArgs struct {
	Port     Port   `json:"port" jsonschema:"tcp port"`
	Data     string `json:"data" jsonschema:"payload"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 or base64"`
}

type sendResult struct {
	Port  Port `json:"port" jsonschema:"tcp port"`
	Bytes int  `json:"bytes" jsonschema:"bytes sent"`
}

type readArgs struct {
	Port     Port   `json:"port" jsonschema:"tcp port"`
	Offset   int64  `json:"offset,omitempty" jsonschema:"read offset"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max bytes to read"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 or base64"`
}

type readResult struct {
	Port      Port   `json:"port" jsonschema:"tcp port"`
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
		Name:    "revshell-listener-mcp",
		Title:   "Reverse-shell listener MCP server",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "status", Description: "Show listeners, connections, and async errors."},
		func(ctx context.Context, req *mcp.CallToolRequest, args statusArgs) (*mcp.CallToolResult, any, error) {
			result := statusResult{
				Listeners: mgr.GetStatus(),
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "listen", Description: "Start listening on a TCP port."},
		func(ctx context.Context, req *mcp.CallToolRequest, args listenArgs) (*mcp.CallToolResult, any, error) {
			l, err := mgr.Listen(args.Port)
			if err != nil {
				return nil, nil, err
			}
			result := listenResult{
				Port:      l.backendPort,
				PublicURL: l.tunnel.URL(),
			}
			return nil, result, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "close", Description: "Stop listening on a TCP port and close connections."},
		func(ctx context.Context, req *mcp.CallToolRequest, args closeArgs) (*mcp.CallToolResult, any, error) {
			if err := mgr.Close(args.Port); err != nil {
				return nil, nil, err
			}
			return nil, closeResult{Port: args.Port}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "send", Description: "Send data to a connection."},
		func(ctx context.Context, req *mcp.CallToolRequest, args sendArgs) (*mcp.CallToolResult, any, error) {
			data, err := decodeData(args.Data, args.Encoding)
			if err != nil {
				return nil, nil, err
			}

			l, ok := mgr.Get(args.Port)
			if !ok {
				return nil, nil, fmt.Errorf("port %d not listening", args.Port)
			}

			if err := l.Send(data); err != nil {
				return nil, nil, err
			}
			return nil, sendResult{Port: args.Port, Bytes: len(data)}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "read", Description: "Read log bytes for a port with an offset/limit for scrolling."},
		func(ctx context.Context, req *mcp.CallToolRequest, args readArgs) (*mcp.CallToolResult, any, error) {
			l, ok := mgr.Get(args.Port)
			if !ok {
				return nil, nil, fmt.Errorf("port %d not listening", args.Port)
			}

			data, next, total, truncated := l.Read(args.Offset, args.Limit)
			encoded, err := encodeData(data, args.Encoding)
			if err != nil {
				return nil, nil, err
			}

			result := readResult{
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
