package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathDefaultsToDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "bram", "config.json")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestPathHonorsXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "bram", "config.json")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestLoadReadsConfiguredPeers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := []byte(`{
  "listen": "127.0.0.1:9000",
  "default_peer": "grok",
  "peers": {
    "grok": {"command": "/bin/cat", "timeout": "5s"}
  }
}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultPeer != "grok" {
		t.Fatalf("DefaultPeer = %q, want grok", cfg.DefaultPeer)
	}
	if cfg.Peers["grok"].Command != "/bin/cat" {
		t.Fatalf("grok command = %q, want /bin/cat", cfg.Peers["grok"].Command)
	}
}
