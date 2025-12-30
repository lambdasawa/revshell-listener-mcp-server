package main

import (
	"os"
	"strconv"

	"github.com/gen2brain/beeep"
)

func sendDesktopNotification(body string) {
	if !desktopNotificationsEnabled() {
		return
	}
	go func() {
		beeep.AppName = "revshell-listener-mcp"
		if err := beeep.Notify("revshell-listener-mcp", body, ""); err != nil {
			// TODO: handle error
		}
	}()
}

func desktopNotificationsEnabled() bool {
	value, ok := os.LookupEnv("REVSHELL_LISTENER_ENABLE_DESKTOP_NOTIFICATION")
	if !ok {
		return true
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return true
	}

	return enabled
}
