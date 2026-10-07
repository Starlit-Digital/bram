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
	if !strings.Contains(body, "mailbox &amp; &lt;example&gt;") || !strings.Contains(body, "ca.starlitdigital.bram") {
		t.Fatalf("invalid mailbox or identity: %s", body)
	}
}

func TestInstallMigratesLegacyAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fake := filepath.Join(home, "bin")
	if err := os.MkdirAll(fake, 0755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(home, "calls")
	t.Setenv("LAUNCHCTL_TEST_LOG", calls)
	t.Setenv("PATH", fake+":"+os.Getenv("PATH"))
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$LAUNCHCTL_TEST_LOG\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(fake, "launchctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agents, 0755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(agents, legacyLabel+".plist")
	if err := os.WriteFile(old, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	path, err := Install()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("legacy plist retained after successful migration")
	}
	body, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(body), label) {
		t.Fatal("new plist missing")
	}
	log, _ := os.ReadFile(calls)
	if !strings.Contains(string(log), "/"+legacyLabel) || !strings.Contains(string(log), "bootstrap ") {
		t.Fatal("legacy job was not transitioned")
	}
}

func TestInstallRestoresLegacyOnBootstrapFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fake := filepath.Join(home, "bin")
	if err := os.MkdirAll(fake, 0755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(home, "calls")
	t.Setenv("LAUNCHCTL_TEST_LOG", calls)
	t.Setenv("PATH", fake+":"+os.Getenv("PATH"))
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$LAUNCHCTL_TEST_LOG\"\ncase \"$*\" in bootstrap*ca.starlitdigital.bram.plist) exit 1;; esac\nexit 0\n"
	if err := os.WriteFile(filepath.Join(fake, "launchctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agents, 0755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(agents, legacyLabel+".plist")
	if err := os.WriteFile(old, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(); err == nil {
		t.Fatal("bootstrap failure not returned")
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("rollback plist removed")
	}
	log, _ := os.ReadFile(calls)
	if !strings.Contains(string(log), "bootstrap gui/") || !strings.HasSuffix(strings.TrimSpace(string(log)), old) {
		t.Fatal("legacy bootstrap not restored")
	}
}
