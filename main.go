package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	sendDesktopNotification("ping")

	mgr := NewListenerManager()
	defer mgr.CloseAll()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		mgr.CloseAll()
	}()

	if err := newMCPServer(mgr).Run(ctx, &mcp.StdioTransport{}); err != nil {
		panic(err)
	}
}
