#!/usr/bin/env python3
import os
import pathlib
import shutil
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parents[1]


def run(args, **kwargs):
    return subprocess.run(args, cwd=ROOT, check=True, text=True, capture_output=True, **kwargs)


def main():
    with tempfile.TemporaryDirectory() as tmp:
        env = os.environ.copy()
        env["PREFIX"] = tmp
        run(["make", "build"], env=env)
        exe = pathlib.Path(tmp) / "bin" / "bram"
        grok = pathlib.Path(tmp) / "bin" / "grok"
        if not exe.exists():
            raise SystemExit("installed bram missing")
        if not grok.exists():
            raise SystemExit("installed grok peer missing")
        version = subprocess.run([str(exe), "version"], check=True, text=True, capture_output=True).stdout.strip()
        if not version:
            raise SystemExit("version output missing")
        receipt = pathlib.Path(tmp) / "share" / "bram" / "install-info.txt"
        receipt_text = receipt.read_text()
        if "binary_sha256:" not in receipt_text:
            raise SystemExit("install receipt missing binary hash")
        if "grok_peer_sha256:" not in receipt_text:
            raise SystemExit("install receipt missing grok peer hash")
        peer_copy = pathlib.Path(tmp) / "share" / "bram" / "peers" / "grok-mailbox.sh"
        if not peer_copy.exists():
            raise SystemExit("share peer copy missing")

        shutil.rmtree(ROOT / ".build", ignore_errors=True)
        run(["make", "compile"], env=env)
        if not (ROOT / ".build" / "bram").exists():
            raise SystemExit("compile-only build missing")

    print("local install checks passed")


if __name__ == "__main__":
    main()
