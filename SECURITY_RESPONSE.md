# Security response: local file paths

## Phase 1: CLI path confinement

- Purpose: prevent direct path traversal in CLI arguments and configured template names.
- Hypothesis: restricting each file or directory argument to a single local path component prevents traversal through `..` and absolute paths while preserving normal commands.
- Scope: `sample` name and template, `config set-template`, and `sync-docs` file arguments.
- Out of scope: changing the configured data root, pre-existing symlinks in the workspace, filesystem race protection, and dependency or Go version upgrades.
- Acceptance: traversal input is rejected before file creation or copying; normal names continue to work; tests exercise each CLI application entry point.
- Later decision: the user approved raising the Go minimum to update dependencies on 2026-10-27.

## Evidence

- Baseline: `go test ./...` passed. `govulncheck -show verbose ./...` found no reachable vulnerable symbols and reported [GO-2026-5970](https://pkg.go.dev/vuln/GO-2026-5970), [GO-2026-6629](https://pkg.go.dev/vuln/GO-2026-6629), and [GO-2026-5024](https://pkg.go.dev/vuln/GO-2026-5024) in imported or required modules. The fixed [`x/text` v0.41.0 module](https://go.googlesource.com/text/+/refs/tags/v0.41.0/go.mod) declares Go 1.25.0.
- Phase 1 verification: `go test ./...` passed after the change. The CLI binary rejected `sample ../outside` with exit code 1 and `invalid issue name`. `git diff --check` passed.

## Phase 2: dependency remediation

- Purpose: clear the three known vulnerability reports from the module graph.
- Hypothesis: updating `golang.org/x/text` to v0.41.0 and `golang.org/x/sys` to v0.44.0 resolves the module reports; Go 1.26.10 also resolves the reachable standard-library finding discovered while testing Go 1.25.0.
- Scope: the Go minimum version, module files, and build requirements in both READMEs.
- Out of scope: replacing CLI dependencies or changing command behavior.
- Acceptance: `go test ./...` and a CLI build pass with Go 1.26.10; `govulncheck ./...` reports no reachable vulnerabilities; module requirements and READMEs agree.
- Later decision: any newly surfaced advisory or incompatible dependency gets its own assessment before further upgrades.
- Verification: Go 1.25.0 tests passed, but `govulncheck@v1.7.0` identified [GO-2026-4602](https://pkg.go.dev/vuln/GO-2026-4602) through `SyncDocs` and `os.ReadDir`. The supported Go 1.26 line fixes it in 1.26.1 and [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809) in 1.26.10. With `GOTOOLCHAIN=go1.26.10`, `go test ./...` passed, all four `build.sh` target combinations built, and `govulncheck@v1.9.0 -show verbose ./...` reported `No vulnerabilities found` across six packages, twelve modules, and the Go 1.26.10 standard library.

## Phase 3: CA certificate advisory in executable artifacts

- Purpose: ensure local executable artifacts are rebuilt with a Go version containing the fix for GO-2026-6809.
- Hypothesis: rebuilding the CLI with Go 1.26.10 and checking build metadata removes the older toolchain from local artifacts. The external `gh` executable must also use a fixed Go version because it performs this CLI's HTTPS requests.
- Boundary: the CLI invokes `gh` through `internal/ghcli/runner.go` and does not import `crypto/x509`, `crypto/tls`, or `net/http` in its executable dependency graph. The installed `gh` binary is the component that verifies GitHub's TLS certificates.
- Scope: the ignored local `gh-issuefs` binary, four ignored `dist/` binaries, the installed `gh issuefs` extension, and verification of the installed `gh` toolchain.
- Out of scope: publishing release assets or changing system certificate stores.
- Acceptance: every local binary and the installed extension report Go 1.26.10 or later, installed `gh` reports Go 1.26.10 or Go 1.27.3 or later, the extension runs, and binary vulnerability scanning finds no reachable vulnerabilities.
- Later decision: publish rebuilt release assets through the release process before using `gh extension upgrade`; a remote upgrade can replace this local repair with the currently published binary.
- Verification: before rebuilding, the root binary, four `dist/` binaries, and installed extension all reported Go 1.26.5; installed `gh` reported Go 1.27.3. Rebuilt the five workspace binaries with Go 1.26.10, backed up the installed extension to `/tmp/gh-issuefs-installed-before-ca-fix`, and installed the rebuilt root binary. Every CLI binary now reports Go 1.26.10. The installed binary and root binary have identical SHA-256 checksums. `gh issuefs config list-templates` completed successfully. `govulncheck@v1.9.0 -mode=binary` on the installed extension reported `No vulnerabilities found`.
