#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
grok mailbox peer for Bram

Reads a prompt from stdin, writes a request file for Grok Bot to answer, waits
for the matching response file, then prints that response to stdout.

Environment:
  BRAM_GROK_MAILBOX          Mailbox root (default: /private/ai-notes/bram)
  BRAM_GROK_TIMEOUT_SECONDS Timeout in seconds (default: 120)
  BRAM_GROK_POLL_SECONDS    Poll interval in seconds (default: 0.2)

Protocol:
  request:  $BRAM_GROK_MAILBOX/inbox/<id>.req.json
  response: $BRAM_GROK_MAILBOX/outbox/<id>.res

The response file is plain text. The first byte written by Grok Bot becomes the
first byte printed by this command.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

if [[ "${1:-}" == "chat" ]]; then
  shift
fi

mailbox="${BRAM_GROK_MAILBOX:-/private/ai-notes/bram}"
timeout_seconds="${BRAM_GROK_TIMEOUT_SECONDS:-120}"
poll_seconds="${BRAM_GROK_POLL_SECONDS:-0.2}"
inbox="$mailbox/inbox"
outbox="$mailbox/outbox"

mkdir -p "$inbox" "$outbox"

prompt="$(cat)"
if [[ -z "$prompt" && "$#" -gt 0 ]]; then
  prompt="$*"
fi
if [[ -z "$prompt" ]]; then
  echo "grok: prompt is required on stdin or argv" >&2
  exit 2
fi

uuid="$(uuidgen | tr '[:upper:]' '[:lower:]')"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
id="${stamp}-${uuid}"
request_path="$inbox/$id.req.json"
response_path="$outbox/$id.res"
tmp_request="$request_path.tmp.$$"

BRAM_GROK_ID="$id" \
BRAM_GROK_REQUEST_PATH="$request_path" \
BRAM_GROK_RESPONSE_PATH="$response_path" \
BRAM_GROK_PROMPT="$prompt" \
python3 - <<'PY' > "$tmp_request"
import json
import os
from datetime import datetime, timezone

payload = {
    "protocol": "bram-grok-mailbox/v1",
    "id": os.environ["BRAM_GROK_ID"],
    "created_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
    "from": "bram",
    "to": "grok",
    "prompt": os.environ["BRAM_GROK_PROMPT"],
    "request_path": os.environ["BRAM_GROK_REQUEST_PATH"],
    "response_path": os.environ["BRAM_GROK_RESPONSE_PATH"],
}
print(json.dumps(payload, indent=2, ensure_ascii=False))
PY
mv "$tmp_request" "$request_path"

deadline="$(python3 - "$timeout_seconds" <<'PY'
import sys
import time
print(time.time() + float(sys.argv[1]))
PY
)"

while true; do
  if [[ -f "$response_path" ]]; then
    cat "$response_path"
    exit 0
  fi

  now="$(python3 - <<'PY'
import time
print(time.time())
PY
)"
  python3 - "$now" "$deadline" <<'PY' || {
import sys
now = float(sys.argv[1])
deadline = float(sys.argv[2])
sys.exit(0 if now <= deadline else 1)
PY
    echo "grok: timed out waiting for $response_path" >&2
    exit 124
  }
  sleep "$poll_seconds"
done

