package launchd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMailboxSelection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("BRAM_GROK_MAILBOX", filepath.Join(home, "custom & mailbox"))
	if got := mailboxPath(home); got != os.Getenv("BRAM_GROK_MAILBOX") {
		t.Fatalf("explicit mailbox lost: %s", got)
	}
	t.Setenv("BRAM_GROK_MAILBOX", "")
	got := mailboxPath(home)
	if info, err := os.Stat("/private/ai-notes/bram"); err == nil && info.IsDir() {
		if got != "/private/ai-notes/bram" {
			t.Fatal("legacy mailbox lost")
		}
	} else if got != filepath.Join(home, ".local", "share", "bram", "mailbox") {
		t.Fatalf("unexpected default: %s", got)
	}
}

func TestPlistEscapesMailboxAndKeepsServiceIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("BRAM_GROK_MAILBOX", "/tmp/mailbox & <example>")
	body, err := Plist()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "mailbox &amp; &lt;example&gt;") || !strings.Contains(body, "ca.simmonsdigitalfoundry.bram") {
		t.Fatalf("invalid mailbox or identity: %s", body)
	}
}
