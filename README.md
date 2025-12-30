# oob-probe-mcp-server

OOB probe MCP server with TCP/HTTP listeners, log access, and ngrok tunnels.

## Features

- TCP tools for reverse shell testing: listen, read logs, and send data to active connections.
- HTTP tools for SSRF testing: receive requests and inspect full request logs.
- ngrok integration for public endpoints on both TCP and HTTP.
- Optional desktop notifications for new connections/requests.

## Tools

TCP:

`listen_tcp`, `send_tcp`, `read_tcp`, `close_tcp`

HTTP:

`listen_http`, `read_http`, `close_http`

Other:

`status`

## Setup

Install the server:

```bash
go install github.com/lambdasawa/oob-probe-mcp-server@latest
```

This server uses ngrok to expose local listeners. Configure an auth token first:

```bash
ngrok config add-authtoken <YOUR_TOKEN>
```

The token is read from your ngrok config file (typically `~/.config/ngrok/ngrok.yml`).

## MCP config example

Example `.mcp.json` entry:

```json
{
  "mcpServers": {
    "oob-probe": {
      "command": "oob-probe-mcp-server",
      "env": {
        "OOB_PROBE_ENABLE_DESKTOP_NOTIFICATION": "false"
      }
    }
  }
}
```

## Desktop notifications

Desktop notifications are enabled by default. Disable them by setting:

```
OOB_PROBE_ENABLE_DESKTOP_NOTIFICATION=false
```

## Intended use

This tool is developed for research purposes in CTFs and controlled lab environments. Use it only against systems and targets you are authorized to test.
