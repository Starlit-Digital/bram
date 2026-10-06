package config

import (
	"encoding/json"
	"github.com/cshaiku/bram/internal/structured"
	"os"
	"path/filepath"
	"testing"
)

func TestGCFConfigAndInvalidProfile(t *testing.T) {
	wire, err := structured.Marshal(Default(), "gcf")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.gcf")
	os.WriteFile(path, wire, 0600)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(cfg)
	b, _ := json.Marshal(Default())
	if string(a) != string(b) {
		t.Fatal("config changed")
	}
	os.WriteFile(path, []byte("GCF profile=graph\n"), 0600)
	if _, err = Load(path); err == nil {
		t.Fatal("accepted graph config")
	}
}
