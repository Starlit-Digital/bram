package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/cshaiku/bram/internal/toolbridge"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/cshaiku/bram/internal/appinfo"
	"github.com/cshaiku/bram/internal/config"
	"github.com/cshaiku/bram/internal/daemon"
	"github.com/cshaiku/bram/internal/launchd"
	"github.com/cshaiku/bram/internal/structured"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if handled, code := toolbridge.Handle(args, "bram", appinfo.Version, stdout, stderr); handled {
		if code != 0 {
			return errors.New("tool workflow failed")
		}
		return nil
	}
	if handled, code := structured.Command(args, "bram", stdout, stderr); handled {
		if code != 0 {
			return errors.New("structured command failed")
		}
		return nil
	}
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
  bram ask [--addr url] [--peer name] [--input FILE|-] [prompt...]
  bram peers [--addr url]
  bram health [--addr url]
  bram sample-config
  bram launchd install|uninstall|plist
  bram tools doctor|plan|run|identity|help
  bram version
  bram capabilities [--format json|gcf|auto]
  bram data encode|decode|stats FILE|- [--format json|gcf|auto]

Ask supports --format text|json|gcf|auto; health and peers support --format json|gcf|auto.
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
	format := fs.String("format", "text", "result format: text, json, gcf or auto")
	input := fs.String("input", "", "append a UTF-8 report from a file or - for stdin (up to 1 MiB)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *format != "text" && !structured.ValidFormat(*format) {
		return errors.New("invalid result format")
	}
	prompt, err := loadPrompt(*input, strings.Join(fs.Args(), " "), os.Stdin)
	if err != nil {
		return err
	}

	requestFormat := *format
	if requestFormat == "text" {
		requestFormat = "json"
	}
	body, err := structured.Marshal(daemon.AskRequest{Peer: *peer, Prompt: prompt}, requestFormat)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(*addr, "/")+"/v1/ask?format="+requestFormat, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	if bytes.HasPrefix(body, []byte("GCF")) {
		req.Header.Set("content-type", "application/gcf")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	wire, err := io.ReadAll(io.LimitReader(resp.Body, structured.MaxBytes+1))
	if err != nil {
		return err
	}
	decoded, err := structured.JSON(wire, structured.MaxBytes)
	if err != nil {
		return err
	}
	var out daemon.AskResponse
	if err := json.Unmarshal(decoded, &out); err != nil {
		return err
	}
	if *format != "text" {
		encoded, err := structured.Marshal(out, *format)
		if err != nil {
			return err
		}
		if _, err = stdout.Write(append(encoded, '\n')); err != nil {
			return err
		}
	}
	if resp.StatusCode >= 400 {
		if out.Error != "" {
			return errors.New(out.Error)
		}
		return fmt.Errorf("daemon returned %s", resp.Status)
	}

	if *format != "text" {
		return nil
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

func loadPrompt(path, prefix string, stdin io.Reader) (string, error) {
	prompt := strings.TrimSpace(prefix)
	if path != "" {
		reader := stdin
		if path != "-" {
			f, err := os.Open(path)
			if err != nil {
				return "", err
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil || !info.Mode().IsRegular() {
				return "", errors.New("prompt input must be a regular file")
			}
			reader = f
		}
		data, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
		if err != nil {
			return "", err
		}
		if len(data) > 1<<20 {
			return "", errors.New("prompt input exceeds 1 MiB")
		}
		if !utf8.Valid(data) {
			return "", errors.New("prompt input is not UTF-8")
		}
		if prompt != "" {
			prompt += "\n\n"
		}
		prompt += strings.TrimSpace(string(data))
	}
	if prompt == "" {
		return "", errors.New("prompt is required")
	}
	return prompt, nil
}

func runPeers(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("peers", flag.ContinueOnError)
	addr := fs.String("addr", "http://127.0.0.1:7878", "daemon URL")
	format := fs.String("format", "json", "result format: json, gcf or auto")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !structured.ValidFormat(*format) {
		return errors.New("invalid result format")
	}
	return getJSON(ctx, strings.TrimRight(*addr, "/")+"/v1/peers", stdout, *format)
}

func runHealth(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	addr := fs.String("addr", "http://127.0.0.1:7878", "daemon URL")
	format := fs.String("format", "json", "result format: json, gcf or auto")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !structured.ValidFormat(*format) {
		return errors.New("invalid result format")
	}
	return getJSON(ctx, strings.TrimRight(*addr, "/")+"/v1/health", stdout, *format)
}

func getJSON(ctx context.Context, url string, stdout io.Writer, format string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"?format="+format, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, structured.MaxBytes+1))
	if err != nil {
		return err
	}
	data, err = structured.JSON(data, structured.MaxBytes)
	if err != nil {
		return err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(&value); err != nil {
		return err
	}
	wire, err := structured.Marshal(value, format)
	if err != nil {
		return err
	}
	_, err = stdout.Write(append(wire, '\n'))
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
