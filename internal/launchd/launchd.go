package launchd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
`, label, xmlEscape(bramPath), xmlEscape(filepath.Join(logDir, "bram.out.log")), xmlEscape(filepath.Join(logDir, "bram.err.log")))
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
