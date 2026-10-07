# Grok adapter handoff

bram is Starlit Digital's 0BSD local command-peer router. README.md owns setup
and behavior; SECURITY.md owns the current trust boundaries; CHANGELOG.md owns
release history. See docs/OPERATIONS.md for the studio checkout and reports.

The supplied grok helper is a mailbox wrapper, not an official provider CLI.
Requests use bram-grok-mailbox/v1 and include id, prompt, request_path and
response_path. A responder should write a complete plain-text response atomically
to that response_path. BRAM_GROK_MAILBOX selects the shared root. Tests use a
mock responder; real-provider qualification remains open. Do not upload prompts
or automatically contact a provider as part of installation or tests.

Possible follow-ups: an independently verified provider adapter, bounded output
and concurrency, API authentication, HTTP handler tests, streaming, redacted
logging and Unix socket transport. None is promised by the 0.2.0 release.
