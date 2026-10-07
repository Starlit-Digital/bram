# bram

A small local router for command-line AI tools, maintained by [Starlit Digital](https://sltd.ca/).

bram runs an HTTP daemon, sends a prompt to a configured command over stdin,
and returns its output, errors, exit code and elapsed time as JSON or GCF. You choose
which commands it can run. It does not include an AI model or provider account.

[Product page](https://sltd.ca/bram/) · [Source](https://github.com/cshaiku/bram) ·
[Changelog](CHANGELOG.md) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)

**Version 0.3.0 · 0BSD · early development · macOS verified.**
The Go router can be built elsewhere, but launchd is macOS-only and the full
installer/mailbox workflow has not been qualified on Linux or Windows.

## Build and try it

Requires Go 1.24.0 or later, Bash, Make and Python 3 for the mailbox helper/tests.

```sh
git clone https://github.com/cshaiku/bram.git
cd bram
make compile
./.build/bram daemon
```

With no configuration file, the daemon listens on `127.0.0.1:7878` and uses
`/bin/cat` as its echo peer. In another terminal:

```sh
./.build/bram health
./.build/bram peers
./.build/bram ask --peer echo "hello from bram"
```

This checks local routing without contacting an AI provider. Stop the foreground
daemon with Ctrl-C.

## Install

```sh
make build                       # compile and install
make install                     # same as make build
make compile                     # compile only to .build/bram
make build PREFIX="$HOME/.local" # explicit installation prefix
```

The default installation copies `bram` and the `grok` mailbox wrapper into
`$HOME/.local/bin`. **It replaces any existing commands with those names.**
Previous commands are retained under `$HOME/.local/share/bram/installs/`.
Use a different PREFIX if you already have another Grok CLI. Add your chosen
prefix's `bin` directory to PATH. The receipt records source, commit, dirty
state, version and executable digests. No repository symlink is installed.

You can install only the Go CLI with
`go install github.com/cshaiku/bram/cmd/bram@v0.3.0`; this does not install the
mailbox wrapper or create an installation receipt.

## Configure peers

Configuration lives at `$XDG_CONFIG_HOME/bram/config.json`, or
`$HOME/.config/bram/config.json` when XDG_CONFIG_HOME is unset. A peer is a
command that reads a prompt on stdin and writes its answer on stdout.

To review the example:

```sh
bram sample-config
```

Save a configuration explicitly; this command only prints the example.
A minimal configuration is:

```json
{
  "listen": "127.0.0.1:7878",
  "default_peer": "echo",
  "peers": {
    "echo": {"command": "/bin/cat", "timeout": "10s"},
    "grok": {"command": "grok", "args": ["chat"], "timeout": "2m"}
  }
}
```

Run `bram daemon --config /path/to/config.json` for a separate configuration.
Peer names, commands and timeouts are validated when the daemon loads the file.
Use trusted commands and keep the listener on loopback. Structured input and output are capped at 64 MiB. The API has no authentication
or concurrency controls, and peer output is still buffered without a subprocess
output limit; see SECURITY.md.

## HTTP API

```sh
curl http://127.0.0.1:7878/v1/health
curl http://127.0.0.1:7878/v1/peers
curl -s http://127.0.0.1:7878/v1/ask \
  -H 'content-type: application/json' \
  -d '{"peer":"echo","prompt":"hello"}'
```

`GET /v1/health` reports name, version and listener. `GET /v1/peers` lists
configured peers. `POST /v1/ask` accepts a prompt and optional peer; a missing
peer selects the default. Success returns `peer`, `output`, optional `stderr`,
`exit_code` and `duration_ms`. A peer execution failure returns HTTP 502 and
an `error`. Validation failures return 400; an unknown peer returns 404.

## GCF interchange

GCF (Generic Context Format) is an alternative structured representation, using
pinned gcf-go v1.8.0. JSON remains the API default and `ask` still prints plain
text by default. The mailbox protocol remains JSON/plain text.

```sh
bram capabilities --format gcf
bram data encode input.json --format gcf
bram data decode input.gcf --format json
bram data stats input.json
bram ask --peer echo --format gcf "hello"
bram health --format gcf
bram peers --format auto
```

`data` accepts a file or `-` for stdin. Encode defaults to GCF; decode, stats and
capabilities default to JSON. `auto` chooses the smaller complete encoding, JSON
on ties, and falls back to JSON for values GCF cannot preserve. Stats compare bytes,
not LLM tokens. Conversion does not run peers or start a daemon.

The API accepts complete JSON or generic-GCF request bodies on the same routes.
Use `Content-Type: application/gcf` for GCF requests. Request GCF responses with
`Accept: application/gcf` or `?format=gcf`; `?format=auto` selects the smaller
encoding. The response Content-Type identifies the selected encoding, including
structured failures. Existing JSON requests/responses remain supported.

Configuration also accepts GCF: use `bram daemon --config config.gcf` explicitly.
Default config lookup still uses config.json. Only complete generic snapshots are
accepted; graph/session-delta profiles and invalid UTF-8/trailing data are rejected.
Input, decoded data and encoded output are limited to 64 MiB. The peer process can
still allocate more output before response admission. GCF does not normalize the
meaning of a provider's text or change command permissions.

## Grok mailbox helper

The bundled `grok` command is a filesystem mailbox adapter, not an official
Grok API integration. It needs a separate responder to read requests and write
responses. Nothing in this release automatically connects to Grok Bot.

```sh
export BRAM_GROK_MAILBOX="$HOME/.local/share/bram/mailbox"
export BRAM_GROK_TIMEOUT_SECONDS=120
export BRAM_GROK_POLL_SECONDS=0.2
```

The mailbox defaults to `$HOME/.local/share/bram/mailbox`. Requests go to
`inbox/<id>.req.json`; the responder writes complete plain text to the
request's `response_path` under `outbox/`. Publish the response atomically after
writing it: the wrapper reads as soon as the file exists. Files contain prompts
and responses and are not automatically removed. The wrapper accepts `grok` and
`grok chat`; `grok --help` describes its protocol. Python 3 and `uuidgen` are
needed. A mock responder is supplied for tests, not real AI answers.

## macOS background service

```sh
bram launchd plist                # review generated configuration
bram launchd install              # install and start the LaunchAgent
bram launchd uninstall            # unload and remove it
```

The agent uses `$HOME/.local/bin/bram`; a custom PREFIX requires your own service
configuration. Logs live under `$HOME/.local/share/bram/logs`. The original
`ca.simmonsdigitalfoundry.bram` service label remains for upgrade compatibility
so a second daemon is not created. Starlit Digital maintains bram.

When generating the agent, an explicit BRAM_GROK_MAILBOX wins. Existing
`/private/ai-notes/bram` directories retain the old mailbox path; new installations
use the home-directory default. Moving the source checkout does not move data,
change existing agent files, or restart a running service.

## Test and release evidence

```sh
go test -count=1 ./...
go vet ./...
python3 scripts/test-local-install.py
python3 scripts/test-grok-mailbox.py
```

The install test uses a temporary prefix; the mailbox test uses a local mock.
They do not exercise real AI providers. Version agreement and launchd mailbox
selection have focused tests. CI runs these checks on macOS; its result must be
checked separately from local evidence. See [release evidence](docs/RELEASE_EVIDENCE.md).

The optional Starlit development collector reads source-bound reports at
`build/release-evidence/latest.json`; it never runs tests. Its three required
IDs are `bram-go-tests`, `bram-local-install` and `bram-grok-mailbox`. Reports
include the commit and complete non-documentation source hash. Source changes
invalidate them. Local setup is described in [operations](docs/OPERATIONS.md).

## License

bram uses the [BSD Zero Clause license (0BSD)](LICENSE), the same license as
Starlit Digital's loom. You may use, copy, modify and distribute it, including
commercially. See the license for its terms and warranty disclaimer.
