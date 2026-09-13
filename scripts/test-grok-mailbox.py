#!/usr/bin/env python3
import os
import pathlib
import subprocess
import tempfile
import time


ROOT = pathlib.Path(__file__).resolve().parents[1]
WRAPPER = ROOT / "scripts" / "peers" / "grok-mailbox.sh"
RESPONDER = ROOT / "scripts" / "peers" / "grok-mailbox-mock-responder.py"


def main():
    with tempfile.TemporaryDirectory() as tmp:
        env = os.environ.copy()
        env["BRAM_GROK_MAILBOX"] = tmp
        env["BRAM_GROK_TIMEOUT_SECONDS"] = "5"
        env["BRAM_GROK_POLL_SECONDS"] = "0.05"

        responder = subprocess.Popen(
            ["python3", str(RESPONDER), "--mailbox", tmp, "--once"],
            cwd=ROOT,
            env=env,
        )
        try:
            result = subprocess.run(
                [str(WRAPPER), "chat"],
                input="hello mailbox",
                text=True,
                capture_output=True,
                cwd=ROOT,
                env=env,
                timeout=10,
                check=True,
            )
        finally:
            try:
                responder.wait(timeout=2)
            except subprocess.TimeoutExpired:
                responder.terminate()
                responder.wait(timeout=2)

        if result.stdout != "mock-grok: hello mailbox\n":
            raise SystemExit(f"unexpected stdout: {result.stdout!r}")

        inbox = pathlib.Path(tmp) / "inbox"
        if len(list(inbox.glob("*.req.json"))) != 1:
            raise SystemExit("request file was not created")

    print("grok mailbox checks passed")


if __name__ == "__main__":
    main()

