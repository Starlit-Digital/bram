# Handoff for Grok: Bram

You are being brought into the Bram project.

## Project

Repo: `/Users/cs/bram`

Bram is a new macOS-local daemon for connecting Codex to Grok or other AI tools.
The original working name was `bridge`; do not use that as the product or repo
name. The name is **Bram** everywhere user-facing.

The design goal is a small, inspectable local service:

- `bram daemon` runs a local HTTP service on `127.0.0.1:7878` by default.
- `bram ask --peer <name> "prompt"` sends prompts through that daemon.
- Peer adapters are configured in `~/.config/bram/config.json`.
- A peer is any command that can receive a prompt on stdin and return text on
  stdout. Grok should be just one peer, not a special-case assumption.
- macOS launchd support should let the daemon run in the background.

## Local tool conventions

Follow the SDF local tool pattern:

- `make build` and `make install` compile and install to `$HOME/.local/bin/bram`.
- `make compile` only writes `.build/bram`.
- Install receipts live in `$HOME/.local/share/bram/install-info.txt`.
- Install history lives under `$HOME/.local/share/bram/installs/`.
- Do not install into Homebrew.
- Do not push, publish, or upload anything unless explicitly asked.
- Preserve unrelated user edits.

Relevant local policy:

- `/Users/cs/.local/share/sdf-tools/README.md`
- `/Users/cs/bram/AGENTS.md`

## Current implementation map

- `cmd/bram/main.go` is the executable entrypoint.
- `internal/app` implements CLI commands.
- `internal/config` loads and validates config.
- `internal/daemon` implements the HTTP API.
- `internal/peer` runs configured peer commands.
- `internal/launchd` writes and loads the macOS LaunchAgent.
- `scripts/build-local.sh` is the local installer.
- `scripts/test-local-install.py` verifies install behavior.

## API

Health:

```sh
curl http://127.0.0.1:7878/v1/health
```

List peers:

```sh
curl http://127.0.0.1:7878/v1/peers
```

Ask a peer:

```sh
curl -s http://127.0.0.1:7878/v1/ask \
  -H 'content-type: application/json' \
  -d '{"peer":"echo","prompt":"hello"}'
```

## Suggested next work

1. Add a real Grok adapter example once the local Grok CLI invocation is known.
2. Decide whether Bram should support streaming responses, probably via SSE.
3. Add request logging with redaction controls.
4. Add an auth token option before exposing anything beyond localhost.
5. Add tests for config validation, peer execution, and HTTP handlers.
6. Consider a Unix socket transport for local tools that should avoid TCP.

## Verification commands

Run from `/Users/cs/bram`:

```sh
go test ./...
python3 scripts/test-local-install.py
make build
~/.local/bin/bram version
```

For a smoke test:

```sh
bram daemon
bram ask --peer echo "hello from Grok"
```

