package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptInputs(t *testing.T) {
	prompt, err := loadPrompt("-", "Explain", strings.NewReader("{\"status\":\"ok\"}"))
	if err != nil || prompt != "Explain\n\n{\"status\":\"ok\"}" {
		t.Fatalf("%q %v", prompt, err)
	}
	p := filepath.Join(t.TempDir(), "report.json")
	os.WriteFile(p, []byte("local report"), 0600)
	prompt, err = loadPrompt(p, "", strings.NewReader("unused"))
	if err != nil || prompt != "local report" {
		t.Fatalf("%q %v", prompt, err)
	}
	for _, input := range [][]byte{bytes.Repeat([]byte("x"), (1<<20)+1), {0xff}, []byte(" ")} {
		if _, err := loadPrompt("-", "", bytes.NewReader(input)); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if _, err := loadPrompt(t.TempDir(), "", strings.NewReader("")); err == nil {
		t.Fatal("directory prompt accepted")
	}
	prompt, err = loadPrompt("", "ordinary prompt", nil)
	if err != nil || prompt != "ordinary prompt" {
		t.Fatal("original prompt behavior changed")
	}
}
