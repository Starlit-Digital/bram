package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cshaiku/bram/internal/appinfo"
	"github.com/cshaiku/bram/internal/config"
	"github.com/cshaiku/bram/internal/daemon"
	"github.com/cshaiku/bram/internal/launchd"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return usage(stdout)
	}

	switch args[0] {
	case "daemon":
		return runDaemon(ctx, args[1:], stdout, stderr)
	case "ask":
		return runAsk(ctx, args[1:], stdout)
	case "peers":
		return runPeers(ctx, args[1:], stdout)
	case "health":
		return runHealth(ctx, args[1:], stdout)
	case "sample-config":
		return sampleConfig(stdout)
	case "launchd":
		return runLaunchd(args[1:], stdout)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, appinfo.Version)
		return nil
	case "help", "--help", "-h":
		return usage(stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage(w io.Writer) error {
	_, err := fmt.Fprint(w, `Bram routes prompts between local AI tools.

Usage:
  bram daemon [--config path] [--listen addr]
  bram ask [--addr url] [--peer name] prompt...
  bram peers [--addr url]
  bram health [--addr url]
  bram sample-config
  bram launchd install|uninstall|plist
  bram version
`)
	return err
}

func runDaemon(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "config path")
	listen := fs.String("listen", "", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = config.Default()
	}
	if *listen != "" {
		cfg.Listen = *listen
	}

	fmt.Fprintf(stdout, "bram daemon listening on %s\n", cfg.Listen)
	return daemon.New(cfg).ListenAndServe(ctx)
}

func runAsk(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	addr := fs.String("addr", "http://127.0.0.1:7878", "daemon URL")
	peer := fs.String("peer", "", "peer name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	prompt := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if prompt == "" {
		return errors.New("prompt is required")
	}

	body, err := json.Marshal(daemon.AskRequest{Peer: *peer, Prompt: prompt})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(*addr, "/")+"/v1/ask", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out daemon.AskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		if out.Error != "" {
			return errors.New(out.Error)
		}
		return fmt.Errorf("daemon returned %s", resp.Status)
	}

	if out.Stderr != "" {
		fmt.Fprintf(stdout, "stderr:\n%s\n", out.Stderr)
	}
	fmt.Fprint(stdout, out.Output)
	if out.Output != "" && !strings.HasSuffix(out.Output, "\n") {
		fmt.Fprintln(stdout)
	}
	return nil
}

func runPeers(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("peers", flag.ContinueOnError)
	addr := fs.String("addr", "http://127.0.0.1:7878", "daemon URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return getJSON(ctx, strings.TrimRight(*addr, "/")+"/v1/peers", stdout)
}

func runHealth(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	addr := fs.String("addr", "http://127.0.0.1:7878", "daemon URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return getJSON(ctx, strings.TrimRight(*addr, "/")+"/v1/health", stdout)
}

func getJSON(ctx context.Context, url string, stdout io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, mustRead(resp.Body), "", "  "); err != nil {
		return err
	}
	pretty.WriteByte('\n')
	_, err = stdout.Write(pretty.Bytes())
	return err
}

func mustRead(r io.Reader) []byte {
	b, err := io.ReadAll(r)
	if err != nil {
		return []byte(`{"error":"read failed"}`)
	}
	return b
}

func sampleConfig(stdout io.Writer) error {
	cfg := config.Sample()
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func runLaunchd(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("launchd requires install, uninstall, or plist")
	}
	switch args[0] {
	case "install":
		path, err := launchd.Install()
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "installed launch agent: %s\n", path)
		return nil
	case "uninstall":
		path, err := launchd.Uninstall()
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "removed launch agent: %s\n", path)
		return nil
	case "plist":
		plist, err := launchd.Plist()
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, plist)
		return nil
	default:
		return fmt.Errorf("unknown launchd action %q", args[0])
	}
}
