# Contributing to bram

bram is maintained by Starlit Digital. Use GitHub issues for bugs and focused
proposals, and pull requests for changes. Security reports belong at security@sltd.ca.

Read README.md, SECURITY.md and AGENTS.md. Use the Go version in go.mod.
Run Go tests/vet and the two Python checks in the README. Keep test providers
local and synthetic; do not require paid AI access. macOS is the verified target.
Do not claim Linux or Windows support based only on compilation.

Keep peer adapters explicit, reviewable and provider-neutral. Preserve existing
configurations, mailbox data and installation history. Document behavior and
limits with each change. VERSION and internal/appinfo must agree; update
CHANGELOG.md and RELEASE_DATE for a release. Contributions are under 0BSD.
