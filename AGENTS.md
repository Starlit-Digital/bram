# Bram repository instructions

Read `README.md` and the relevant docs/source before changing behavior. Preserve
pre-existing working-tree changes. This repo may contain another task's edits.

## Build and install contract

- The standard native developer command is `make build` (alias `make install`).
  A successful build installs `bram` into `$HOME/.local/bin`; it is not a
  repository symlink. Supporting files and provenance live under
  `$HOME/.local/share/bram`.
- `make compile` only writes `.build/bram`. Use this for CI, cross-compilation,
  or when an installation is not authorized. Raw `go build` does not install.
- Override with `make build PREFIX=/absolute/prefix`; never hardcode Homebrew's
  prefix or assume sudo. Use Go requirements from `go.mod`, including toolchain.
- `scripts/build-local.sh` is the checked-in installation implementation. It
  refuses to install foreign-architecture binaries, stages before replacement,
  and retains previous binaries. Do not clean install history automatically.
- After build, verify `command -v bram` and its documented read-only help/version
  from outside the checkout.
- Run `go test ./...` for Go changes and `python3 scripts/test-local-install.py`
  for installation changes. Avoid invoking network/AI/upload operations for a
  startup smoke.
- User-local PATH must precede Homebrew for our own tools. This Mac's shared PATH
  helper is `~/.config/sdf/tool-path.sh`; launchd applies it to newly launched apps.
  Other machines need `$HOME/.local/bin` on PATH. Do not assume the Mac helper exists.
- Source dumps use the global `sourcedump` command; do not upload or publish source
  as part of building or installing.

