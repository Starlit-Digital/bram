#!/usr/bin/env python3
"""Small test responder for the Bram/Grok mailbox protocol."""

import argparse
import json
import pathlib
import time


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mailbox", default="/private/ai-notes/bram")
    parser.add_argument("--once", action="store_true")
    parser.add_argument("--prefix", default="mock-grok: ")
    parser.add_argument("--poll", type=float, default=0.1)
    args = parser.parse_args()

    root = pathlib.Path(args.mailbox)
    inbox = root / "inbox"
    outbox = root / "outbox"
    inbox.mkdir(parents=True, exist_ok=True)
    outbox.mkdir(parents=True, exist_ok=True)

    seen: set[pathlib.Path] = set()
    while True:
        handled = False
        for path in sorted(inbox.glob("*.req.json")):
            if path in seen:
                continue
            seen.add(path)
            request = json.loads(path.read_text())
            response_path = pathlib.Path(request["response_path"])
            response_path.parent.mkdir(parents=True, exist_ok=True)
            response_path.write_text(f"{args.prefix}{request['prompt']}\n")
            handled = True
            if args.once:
                return 0
        if args.once and handled:
            return 0
        time.sleep(args.poll)


if __name__ == "__main__":
    raise SystemExit(main())

