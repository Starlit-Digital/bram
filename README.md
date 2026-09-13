# Bram

**Bram** is a local macOS daemon for routing prompts and responses between AI
tools. It is intended to let Codex talk to Grok, or any other command-line AI
tool, through one small, auditable local service.

The name replaces the working title `bridge`. Bram is the product/repository
name; bridging is only the job it performs.

## What it does

- Runs a local HTTP daemon on `127.0.0.1:7878` by default.
- Loads peer adapters from `~/.config/bram/config.json`.
- Sends prompts to a configured peer command over stdin.
- Returns structured JSON with stdout, stderr, exit code, duration, and errors.
- Provides a small CLI for health checks, peer listing, direct asks, and launchd
  installation.
- Installs locally under `$HOME/.local`, matching the SDF tool convention.

## Quick start

```sh
make build
bram sample-config
bram daemon
```

In another terminal:

```sh
bram ask --peer echo "hello from Codex"
```

## Configuration

Create `~/.config/bram/config.json`:

```json
{
  "listen": "127.0.0.1:7878",
  "default_peer": "grok",
  "peers": {
    "grok": {
      "command": "grok",
      "args": ["chat"],
      "timeout": "2m"
    },
    "echo": {
      "command": "/bin/cat",
      "timeout": "10s"
    }
  }
}
```

Every peer command receives the prompt on stdin. This keeps Bram provider-neutral:
Grok, Claude, local Ollama wrappers, shell scripts, and future AI tools all fit
behind the same contract.

## HTTP API

```sh
curl http://127.0.0.1:7878/v1/health
curl http://127.0.0.1:7878/v1/peers
curl -s http://127.0.0.1:7878/v1/ask \
  -H 'content-type: application/json' \
  -d '{"peer":"echo","prompt":"hello"}'
```

## macOS launchd

```sh
bram launchd install
bram launchd uninstall
```

The launch agent runs `bram daemon` and writes logs under
`~/.local/share/bram/logs`.

## Local developer installation

```sh
make build                       # compile and install ~/.local/bin/bram
make compile                     # compile only to .build/bram
make build PREFIX="$HOME/.local" # explicit installation prefix
```

`make install` is equivalent to `make build`. The installed executable is copied
out of the checkout, so moving the source repo does not break it.

## Testing

```sh
go test ./...
python3 scripts/test-local-install.py
```

