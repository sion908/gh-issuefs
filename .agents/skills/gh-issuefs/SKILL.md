---
name: gh-issuefs
description: Use gh issuefs to work with GitHub Issues and pull requests as local Markdown files. Apply when a task involves pulling, editing, creating, or pushing issues in a repository that uses gh-issuefs.
---

# gh-issuefs

Use `gh issuefs <command> --help` for current syntax. Check `.design/config.toml` when present: it can change the local data directories and the default pull query.

## Read or edit an existing issue

1. Run `gh issuefs pull <number>` to fetch the issue. This writes local files. If a previously synced `issue.md` had local changes, the command moves it to an `issue.local*.md` backup; inspect that backup before editing.
2. Find the numbered directory under the configured issues directory and read `issue.md`. Pull requests go under the configured PR directory as `pr.md` and are read-only for push.
3. Edit the issue title in YAML frontmatter or the Markdown body. `push` sends only title and body; changes to other frontmatter fields are not synced.
4. Before an authorized remote update, inspect the actual file changes and run `gh issuefs push --dry-run <number>`. The preview identifies candidates but does not show a full body diff. Push does not detect newer remote edits, so check the current remote issue when working from an older draft. Then run `gh issuefs push <number>` if the requested update is ready.

## Create an issue

Run `gh issuefs sample <name>`, edit the new `issue.md`, and inspect all unnumbered issue directories before `gh issuefs push --new`. That command creates a GitHub issue for **every** unnumbered directory; it does not accept a single-directory target. `gh issuefs push --new --dry-run` previews those candidates.

## Interpret command results

- Prefer numbered `pull` and `push` commands when the task concerns specific issues. Without numbers, `pull` uses the configured query and `push` scans all changed numbered issue directories.
- `pull`, `push`, and `push --new` can print per-item failures to stderr while exiting with status 0. Check stderr and the output or affected files before reporting success.
- `gh issuefs config set-template <filename>` is non-interactive. Omitting the filename opens a prompt.
- `sync-docs --remove` deletes files from the local docs directory. Inspect its target set before using it.
