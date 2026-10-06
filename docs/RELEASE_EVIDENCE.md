# Release evidence — 0.2.0

2026-10-06, native macOS arm64, Go requirement 1.24.0 (go.mod).

- Fresh `go test -count=1 ./...` passed: config, peer execution,
  release version agreement and LaunchAgent mailbox/identity/XML behavior.
- `go vet ./...` passed.
- Temporary-prefix installer check passed: Bram/helper copies, receipts,
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
