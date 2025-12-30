package main

import (
	"context"
	"fmt"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ngrokConfig struct {
	Agent struct {
		Authtoken string `yaml:"authtoken"`
	} `yaml:"agent"`
}

func (l *Listener) StartNgrokTunnel() error {
	ctx := context.Background()

	backendURL, err := url.Parse(fmt.Sprintf("tcp://127.0.0.1:%d", l.backendPort))
	if err != nil {
		return fmt.Errorf("failed to parse backend URL: %v", err)
	}

	tunnelConfig := config.TCPEndpoint()

	token, err := ngrokAuthtokenFromConfig()
	if err != nil {
		return fmt.Errorf("failed to get ngrok authtoken from config: %v", err)
	}

	tunnel, err := ngrok.ListenAndForward(ctx, backendURL, tunnelConfig, ngrok.WithAuthtoken(token))
	if err != nil {
		return fmt.Errorf("failed to start ngrok tunnel: %v", err)
	}

	l.tunnel = tunnel

	return nil
}

func ngrokAuthtokenFromConfig() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "ngrok", "ngrok.yml")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read ngrok config file %s: %v", configPath, err)
	}

	var cfg ngrokConfig
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse ngrok config file %s: %v", configPath, err)
	}

	token := strings.TrimSpace(cfg.Agent.Authtoken)
	if token == "" {
		return "", fmt.Errorf("ngrok authtoken missing in %s", configPath)
	}
	return token, nil
}
