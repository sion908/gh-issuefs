# gh-issuefs

GitHub Issue をローカルファイルに同期し、設計メモやタスク管理を行うための [GitHub CLI](https://cli.github.com/) extension です。

> **Also available in:** [English](./README.md)

---

## 経緯

このツールは GitHub Issue をローカルの Markdown ファイルとして管理するために作成されました。以下の目的があります：

- **AI 対応**: YAML frontmatter 付きの構造化された Markdown は、AI アシスタントが読み取りやすく理解しやすい
- **編集可能**: ローカルで Issue を修正し、変更を GitHub に反映できる
- **計画駆動**: ローカルの Issue ファイルを使って詳細な実装計画（例: `.design/plan.md`）を作成できる

Issue をローカルファイルとして保持することで、AI ツールを活用して計画・ドキュメント・コード生成を行いながら、GitHub を単一の真実の情報源として維持できます。

---

## 概要

`gh-issuefs` は GitHub Issue を `.design/issues/` 配下のローカル Markdown ファイルとして同期します。  
各 Issue は YAML frontmatter 付きの `issue.md` として保存されるため、コードと並べて編集・レビュー・管理が容易になります。

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

## 必要なもの

- [GitHub CLI](https://cli.github.com/) (`gh`) — 認証済みであること
- Go 1.22 以上（ソースからビルドする場合のみ）

---

## インストール

```bash
gh extension install sion908/gh-issuefs
```

### ソースからビルド

```bash
git clone https://github.com/sion908/gh-issuefs
cd gh-issuefs
go build -o gh-issuefs ./cmd/gh-issuefs
gh extension install .
```

---

## コマンド

### `init`

現在のリポジトリで `gh-issuefs` を初期化します。

```bash
gh issuefs init
gh issuefs init --owner myorg --repo myrepo
```

`.design/config.toml` をデフォルト設定で作成します。

---

### `pull`

GitHub から Issue をローカルの `.design/issues/` に同期します。

```bash
# config.toml のデフォルトクエリで pull
gh issuefs pull

# 番号を指定して pull
gh issuefs pull 123
gh issuefs pull 123 124 125

# 生の API レスポンスも保存
gh issuefs pull --save-raw
```

- Pull Request も pull できます（読み取り専用）。
- `issue.md` にローカルの未同期変更がある場合、上書き前に自動的に `issue.local.md` としてバックアップされます。

---

### `push`

ローカルの `issue.md` の変更を GitHub に反映します。

```bash
# 変更されたすべての Issue を push
gh issuefs push

# 指定した Issue を push
gh issuefs push 123

# 変更内容を確認するだけで実際には push しない
gh issuefs push --dry-run

# 番号未設定のディレクトリから新規 Issue を作成
gh issuefs push --new
```

- Pull Request は push できません（読み取り専用）。

---

### `sample`

テンプレートから新しい Issue ディレクトリを生成します。

```bash
gh issuefs sample add_login_error_handling
gh issuefs sample add_login_error_handling --template default.md
```

`.design/issues/add_login_error_handling/issue.md` を作成します。  
`config.toml` に `default_template` が設定されている場合は自動的に使用されます。

---

### `config`

設定を管理します。

```bash
# 現在の設定を表示
gh issuefs config

# .github/ISSUE_TEMPLATE/ のテンプレート一覧を表示
gh issuefs config list-templates

# インタラクティブにデフォルトテンプレートを選択（矢印キー + Enter）
gh issuefs config set-template

# デフォルトテンプレートを直接指定
gh issuefs config set-template default.md
```

---

## 設定ファイル

`init` 時に `.design/config.toml` が作成されます。

```toml
default_template = ""   # .github/ISSUE_TEMPLATE/ のデフォルトテンプレート

[data]
root_dir   = ".design"
issues_dir = "issues"

[sync]
default_query        = "assignee:@me state:open"
include_comments     = true
exclude_pull_requests = false

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

## issue.md のフォーマット

```markdown
---
number: 123
title: ログインエラーハンドリングの追加
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

## 概要

## 背景

## やること
- [ ]

## 完了条件
- [ ]
```

---

## ライセンス

MIT

---

## ワークフロー

AI アシスト付きのワークフローについては [WORKFLOWS.md](./WORKFLOWS.md) を参照してください。
