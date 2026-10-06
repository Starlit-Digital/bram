package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cshaiku/bram/internal/structured"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Listen      string          `json:"listen"`
	DefaultPeer string          `json:"default_peer,omitempty"`
	Peers       map[string]Peer `json:"peers"`
}

type Peer struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Timeout string   `json:"timeout,omitempty"`
}

func Default() Config {
	return Config{
		Listen: "127.0.0.1:7878",
		Peers: map[string]Peer{
			"echo": {Command: "/bin/cat", Timeout: "10s"},
		},
		DefaultPeer: "echo",
	}
}

func Sample() Config {
	cfg := Default()
	cfg.DefaultPeer = "grok"
	cfg.Peers["grok"] = Peer{Command: "grok", Args: []string{"chat"}, Timeout: "2m"}
	return cfg
}

func Load(path string) (Config, error) {
	if path == "" {
		var err error
		path, err = Path()
		if err != nil {
			return Config{}, err
		}
	}

	b, err := structured.ReadFile(path, structured.MaxBytes)
	if err != nil {
		return Config{}, err
	}

	b, err = structured.JSON(b, structured.MaxBytes)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	return cfg, nil
}

func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "bram", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "bram", "config.json"), nil
}

func (c Config) Validate() error {
	if c.Listen == "" {
		return errors.New("listen is required")
	}
	if len(c.Peers) == 0 {
		return errors.New("at least one peer is required")
	}
	for name, peer := range c.Peers {
		if name == "" {
			return errors.New("peer name cannot be empty")
		}
		if peer.Command == "" {
			return fmt.Errorf("peer %q command is required", name)
		}
		if peer.Timeout != "" {
			if _, err := time.ParseDuration(peer.Timeout); err != nil {
				return fmt.Errorf("peer %q timeout: %w", name, err)
			}
		}
	}
	if c.DefaultPeer != "" {
		if _, ok := c.Peers[c.DefaultPeer]; !ok {
			return fmt.Errorf("default peer %q is not configured", c.DefaultPeer)
		}
	}
	return nil
}
