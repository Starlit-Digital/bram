# Studio operations

Canonical checkout: `/private/var/www/starlit-digital/bram` (moved 2026-10-06).
The previous `/Users/cs/bram` directory is not the source location anymore.
The move retained Git history, existing edits, ignored outputs and Finder metadata.
No compatibility symlink is required: installed tools are copied executables.

The account-local tool registry and SLTD collector configuration now use this
path. Existing installation receipts remain historical; do not rewrite them to
claim a fresh installation. Rebuild through `make build` when installing an update.
Moving source does not move settings/mailboxes or restart launchd.

## Source-bound reports

From this checkout, explicitly run each check through the studio runner:

```sh
python3 ../sltd.ca/tools/dev-signals/run-check.py --repo . --id bram-go-tests -- go test -count=1 ./...
python3 ../sltd.ca/tools/dev-signals/run-check.py --repo . --id bram-local-install -- python3 scripts/test-local-install.py
python3 ../sltd.ca/tools/dev-signals/run-check.py --repo . --id bram-grok-mailbox -- python3 scripts/test-grok-mailbox.py
```

These reports are ignored local output. They require the sibling maintained
website checkout and are optional for public contributors. Collection only reads
reports and uploads bounded metadata; it never runs checks or uploads source.
Renew after a release commit so the receipt matches that commit and source hash.
