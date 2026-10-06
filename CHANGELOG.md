# Changelog

## 0.2.0 — 2026-10-06

First public Starlit Digital release, licensed under 0BSD to match Loom.

- Publish the source as github.com/cshaiku/bram and use its canonical Go module path.
- Identify Starlit Digital as maintainer; add license, contribution and security docs.
- Add VERSION and RELEASE_DATE; include owner/version in installation receipts.
- Move the studio checkout under starlit-digital and update development collection.
- Default new mailbox helpers to the user's home directory. Explicit paths and
  legacy LaunchAgent mailbox directories remain supported.
- Retain the original LaunchAgent label for upgrade compatibility.
- Correct the quick start: sample-config prints JSON and does not save it.
- Add version/launchd tests and a macOS CI workflow; record scoped release evidence.
- Add the Bram product page and product listing on sltd.ca.

Existing local settings, mailbox files and running services are not migrated or
restarted by moving the source. Remote-provider integration, streaming, API
authentication, resource bounds and native Linux/Windows qualification remain open.

## 0.1.0 — 2026-09-13

Local development foundation: HTTP health/peer/ask routes, command peers with
stdin/stdout and timeouts, macOS launchd support, user-local installation receipts,
Grok mailbox helper, configuration and peer tests, and local mock/install checks.
