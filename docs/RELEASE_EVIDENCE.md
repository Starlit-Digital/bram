# Release evidence — 0.2.0

2026-10-06, native macOS arm64, Go requirement 1.24.0 (go.mod).

- Fresh `go test -count=1 ./...` passed: config, peer execution,
  release version agreement and LaunchAgent mailbox/identity/XML behavior.
- `go vet ./...` passed.
- Temporary-prefix installer check passed: bram/helper copies, receipts,
  version/help behavior and compile-only lane.
- Mock mailbox check passed with an isolated temporary directory.
- An isolated daemon on a temporary loopback port passed HTTP health/version,
  peers, ask and CLI ask against `/bin/cat`. The process was then stopped.
- Whitespace checks passed. The five existing Git commits were checked for common
  credential patterns with no matches; this is not an exhaustive security audit.

No real AI provider was contacted. No existing daemon was restarted, live mailbox
moved or user configuration changed. Several packages have no unit tests; the
HTTP smoke covers the happy echo path rather than full API error/resource behavior.
Linux/Windows native workflows, authenticated/remote service exposure, streaming,
resource controls and real-provider compatibility are not qualified.

GitHub CI is configured for macOS. Local passes do not establish its result;
inspect the workflow run associated with the released commit. No binary artifacts
are supplied by this source release. Native installation uses make build.

Source-bound studio receipts are retained locally under build/release-evidence.
They are renewed after committing the release and must match the current source.
Starlit website claims are limited to these capabilities and checks. README,
Changelog, license, security, contribution and operations docs were compared with
the live source for the public release.

## 0.3.0 GCF source verification — 2026-10-06

Fresh local source passed `go test -count=1 ./...`, `go vet ./...`, the
Python local installer checks, and `git diff --check` on this Mac.
Fresh compile-only binaries exchanged generic GCF in both directions between
bram and loom, preserving Unicode, a numeric string, and an integer above
JavaScript’s exact-number range. Capability output matched VERSION.

Tests cover invalid profiles and input bounds. bram additionally passed the
mailbox helper checks and GCF HTTP success/error and config tests. loom tests
cover report equivalence, manifest/profile inputs and overwrite protection;
`verify --json` and `checks:command-catalog --json` passed.

These are source checks, not a tagged binary release or current Linux/Windows
qualification. No real provider or live daemon was used. README, changelog,
interface documentation and third-party licensing were compared with live source.
The existing website release links still refer to the previously tagged releases.

## Documentation naming sweep — 2026-10-06

Product display names use lowercase loom, nora, bram and clyde. The four tracked
documentation trees, issue templates, textual help images and clyde manual were
checked. Technical environment-variable names, Go identifiers and the loom
LOOM-BEGIN/LOOM-END source markers retain their executable spelling. nora's
tracked documentation already matched. This changes presentation only; existing
release versions, behavior, feature/license promises and platform evidence remain
applicable. clyde's manual generation/check, GitHub policy check and SVG XML
validation passed. No new runtime release or installer change is claimed.

## 0.4.0 optional integration source verification — 2026-10-07

All six participating CLIs passed their Go suites and vet checks on macOS arm64.
bram's race checks passed for app, daemon, peer and toolbridge. Native make build
installed the selected sources. Identity, missing-companion doctor and original
native discovery commands passed from outside the source checkouts.

Native repository/UI/local-log recipes passed against synthetic fixtures, with
optional private Vigil health preview leaving the inspected repository unchanged.
Reports preserved private permissions, exact stdout hashes and anonymized log IPs.
Failed fixture reports were retained and other steps continued. An isolated daemon
with /bin/cat passed stdin input and explicit report feedback; no real AI provider,
SSH or upload operation was used. Existing daemons/configuration were not changed.
The selected private Vigil needed a local wrapper persistence fix for health
previews; that private source is not part of these public snapshots.

Evidence is retained locally under
/private/var/www/starlit-digital/archives/starlit-cli-integration-20261007.
These checks do not establish tagged-release, native Linux/Windows, complete
process isolation or real-provider compatibility. Peer bounds/concurrency limits
apply to newly started updated daemons; installing does not restart existing ones.
