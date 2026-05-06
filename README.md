# gh-issuefs

A [GitHub CLI](https://cli.github.com/) extension to sync GitHub Issues to local files for design notes and task management.

> **Also available in:** [日本語](./README.ja.md)

---

## Overview

`gh-issuefs` keeps GitHub Issues in sync with local Markdown files under `.design/issues/`.  
Each issue is stored as `issue.md` with YAML frontmatter, making it easy to edit, review, and track alongside your code.

```
.design/
  config.toml
  issues/
    123_add_login_error_handling/
      issue.md
      comments.json
      .meta.json
```

---

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) — must be authenticated
- Go 1.22+ (only needed to build from source)

---

## Installation

```bash
gh extension install sion908/gh-issuefs
```

### Build from source

```bash
git clone https://github.com/sion908/gh-issuefs
cd gh-issuefs
go build -o gh-issuefs ./cmd/gh-issuefs
gh extension install .
```

---

## Commands

### `init`

Initialize `gh-issuefs` in the current repository.

```bash
gh issuefs init
gh issuefs init --owner myorg --repo myrepo
```

Creates `.design/config.toml` with default settings.

---

### `pull`

Pull issues from GitHub into `.design/issues/`.

```bash
# Pull issues matching default query in config.toml
gh issuefs pull

# Pull specific issues by number
gh issuefs pull 123
gh issuefs pull 123 124 125

# Save raw API responses
gh issuefs pull --save-raw
```

- If `issue.md` has local uncommitted changes, they are automatically backed up to `issue.local.md` before overwriting.

---

### `push`

Push local `issue.md` changes to GitHub.

```bash
# Push all changed issues
gh issuefs push

# Push specific issues
gh issuefs push 123

# Preview changes without pushing
gh issuefs push --dry-run

# Create new issues from unnumbered directories
gh issuefs push --new
```

---

### `sample`

Generate a new issue directory from a template.

```bash
gh issuefs sample add_login_error_handling
gh issuefs sample add_login_error_handling --template default.md
```

Creates `.design/issues/add_login_error_handling/issue.md`.  
If `default_template` is set in `config.toml`, it is used automatically.

---

### `config`

Manage configuration.

```bash
# Show current settings
gh issuefs config

# List available templates from .github/ISSUE_TEMPLATE/
gh issuefs config list-templates

# Set default template interactively (arrow keys + enter)
gh issuefs config set-template

# Set default template directly
gh issuefs config set-template default.md
```

---

## Configuration

`config.toml` is created at `.design/config.toml` on `init`.

```toml
default_template = ""   # default template from .github/ISSUE_TEMPLATE/

[data]
root_dir   = ".design"
issues_dir = "issues"

[sync]
default_query        = "assignee:@me state:open"
include_comments     = true
exclude_pull_requests = true

[storage]
save_raw = false

[formatter]
include_labels      = true
include_milestone   = true
include_assignees   = true
include_projects_v2 = true

[push]
issue_title = true
issue_body  = true
```

---

## issue.md format

```markdown
---
number: 123
title: Add login error handling
state: open
labels:
  - bug
assignees:
  - sion908
milestone: null
projects_v2:
  - name: My Project
    status: In Progress
---

## Overview

## Background

## Tasks
- [ ]

## Acceptance Criteria
- [ ]
```

---

## License

MIT
