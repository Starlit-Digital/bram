# Changelog

## 0.3.1 — 2026-10-06

- Migrate launchd installation to ca.starlitdigital.bram, preserving mailbox settings and restoring the legacy job if replacement bootstrap fails.
- Use the canonical Starlit Mac tool configuration path.

## 0.3.0 — 2026-10-06

- Add bounded JSON/generic-GCF conversion, byte stats and discovery using pinned
  Apache-2.0 gcf-go v1.8.0, with dependency notices and its license.
- Add CLI result formats to ask/health/peers, retaining plain ask and JSON defaults.
- Accept GCF configuration and API requests; negotiate GCF/Auto responses and
  structured errors using Accept or the format query.
- Reject invalid UTF-8, trailing data, stateful profiles and oversized wire data.
- Add API success/error/JSON compatibility and codec round-trip/limit tests.
- Preserve peer execution, mailbox JSON/plain text, config paths and service identity.


## 0.2.0 — 2026-10-06

First public Starlit Digital release, licensed under 0BSD to match loom.

- Publish the source as github.com/cshaiku/bram and use its canonical Go module path.
- Identify Starlit Digital as maintainer; add license, contribution and security docs.
- Add VERSION and RELEASE_DATE; include owner/version in installation receipts.
- Move the studio checkout under starlit-digital and update development collection.
- Default new mailbox helpers to the user's home directory. Explicit paths and
  legacy LaunchAgent mailbox directories remain supported.
- Retain the original LaunchAgent label for upgrade compatibility.
- Correct the quick start: sample-config prints JSON and does not save it.
- Add version/launchd tests and a macOS CI workflow; record scoped release evidence.
- Add the bram product page and product listing on sltd.ca.

Existing local settings, mailbox files and running services are not migrated or
restarted by moving the source. Remote-provider integration, streaming, API
authentication, resource bounds and native Linux/Windows qualification remain open.

## 0.1.0 — 2026-09-13

Local development foundation: HTTP health/peer/ask routes, command peers with
stdin/stdout and timeouts, macOS launchd support, user-local installation receipts,
Grok mailbox helper, configuration and peer tests, and local mock/install checks.
