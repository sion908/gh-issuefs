# AI-facing CLI review

## Phase 1: discoverable guidance

- Purpose: let an AI agent discover the supported `gh issuefs` workflow and distinguish local writes from GitHub writes.
- Hypothesis: a tracked repository skill and accurate CLI help reduce incorrect commands and unsafe broad pushes.
- Scope: a repository-local skill, command and option help, README links to the skill, and contradictory dry-run output found during CLI smoke testing.
- Out of scope: changing sync or push semantics, machine-readable output, and installing skills into other repositories.
- Acceptance: the skill validates as a skill package; `--help` explains defaults, side effects, and important limits; documented `config` usage works; dry-run output does not claim there is nothing to push when a candidate exists; source tests and representative CLI commands pass.
- Later decision: choose whether the CLI should generate a skill in target repositories and whether partial per-item failures should cause nonzero exit status.

## Baseline findings

- The existing `.design/skill/SKILL.md` is excluded from Git and its `.sample` symlink points to an old absolute path, so neither is a distributable entry point.
- `gh issuefs config` is documented but fails because the parser requires a subcommand.
- `--help` lists syntax but does not explain the default query, local writes, remote writes, or the broad scope of `push --new`.
- `pull`, `push`, and `push --new` log some per-item failures to stderr yet return success. Agents must inspect output until the exit-status contract is addressed.

## Verification

- Added a tracked `.agents/skills/gh-issuefs/SKILL.md` and narrowed `.gitignore` so the skill is included in Git. The `skill-creator` validator reported `Skill is valid!`.
- Built and installed the CLI, then checked top-level, `pull`, `push`, `config set-template`, and `sync-docs` help. `gh issuefs config` now prints `default_template` with exit status 0. The installed extension and root binary have matching SHA-256 checksums and report Go 1.26.10.
- `go test ./...` passed after the parser change. The remaining partial-failure exit-status behavior is documented in the skill and remains a later contract decision.
- An isolated CLI run with a fake `gh` command completed `init`, `sample`, `pull 123`, `push --dry-run 123`, and `push --new --dry-run` without a remote write or unexpected `gh` command. The first run revealed that `push --dry-run 123` printed both a preview candidate and `nothing to push`. After the correction, it prints only the candidate; the new-issue preview no longer says `creating issue`.
- `GOTOOLCHAIN=go1.26.10 go test ./...` passed after the correction. The updated installed extension reports Go 1.26.10 and has the same SHA-256 checksum as the rebuilt root binary. The final skill passed `skill-creator` validation.

## Assessment and next phases

The CLI now exposes the basic workflow to an agent, but unattended remote writes remain hard to judge reliably:

| Priority | Evidence | Proposed follow-up and decision |
| --- | --- | --- |
| High | `pull`, `push`, and `push --new` log per-item errors but can exit 0. | Define an aggregate result contract. Compare nonzero exit on any item failure with preserving the current exit code plus a machine-readable summary; assess existing script compatibility before changing it. |
| High | `push --new` scans every unnumbered directory; it cannot target one draft. | Compare a positional directory selector with a new per-draft command. Acceptance should prove that selecting one draft creates only that issue. |
| High | `push` sends the local title and body without checking for newer remote edits. | Define a conflict check using the last synced remote state and a deliberate override path. Acceptance should prove stale drafts do not silently overwrite newer GitHub content. |
| Medium | `push --dry-run` says the body changed but does not show a full diff. | Add a reviewable preview, with clear treatment of large bodies and metadata fields that are not pushed. |
| Medium | The skill is only discoverable in this source repository until copied elsewhere. | If per-repository generation is wanted, choose the target agent locations and an update policy before adding a generator to the CLI. |
