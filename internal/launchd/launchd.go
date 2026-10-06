package launchd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Retained for upgrade compatibility; product ownership is Starlit Digital.
const label = "ca.simmonsdigitalfoundry.bram"

func Install() (string, error) {
	plist, err := Plist()
	if err != nil {
		return "", err
	}
	path, err := plistPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return "", err
	}
	_ = exec.Command("launchctl", "bootout", "gui/"+fmt.Sprint(os.Getuid()), path).Run()
	if err := exec.Command("launchctl", "bootstrap", "gui/"+fmt.Sprint(os.Getuid()), path).Run(); err != nil {
		return path, err
	}
	return path, nil
}

func Uninstall() (string, error) {
	path, err := plistPath()
	if err != nil {
		return "", err
	}
	_ = exec.Command("launchctl", "bootout", "gui/"+fmt.Sprint(os.Getuid()), path).Run()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return path, err
	}
	return path, nil
}

func Plist() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	bramPath := filepath.Join(home, ".local", "bin", "bram")
	logDir := filepath.Join(home, ".local", "share", "bram", "logs")
	pathEnv := strings.Join([]string{
		filepath.Join(home, ".local", "bin"),
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/go/bin",
		filepath.Join(home, "go", "bin"),
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
		"/opt/homebrew/opt/llvm/bin",
	}, ":")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>daemon</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>%s</string>
    <key>BRAM_GROK_MAILBOX</key>
    <string>%s</string>
    <key>BRAM_GROK_POLL_SECONDS</key>
    <string>0.1</string>
    <key>BRAM_GROK_TIMEOUT_SECONDS</key>
    <string>120</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`, label, xmlEscape(bramPath), xmlEscape(pathEnv), xmlEscape(mailboxPath(home)), xmlEscape(filepath.Join(logDir, "bram.out.log")), xmlEscape(filepath.Join(logDir, "bram.err.log")))
	return buf.String(), nil
}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func xmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return replacer.Replace(s)
}

// Existing private-path mailboxes remain usable on upgraded installations.
func mailboxPath(home string) string {
	if value := os.Getenv("BRAM_GROK_MAILBOX"); value != "" {
		return value
	}
	legacy := "/private/ai-notes/bram"
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return legacy
	}
	return filepath.Join(home, ".local", "share", "bram", "mailbox")
}
