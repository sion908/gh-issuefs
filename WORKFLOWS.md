# Workflows

このディレクトリには `gh-issuefs` の使用方法を説明するワークフローが含まれています。各ワークフローは `.windsurf/workflows/` に保存されており、AI アシスタントが直接参照して実行できます。

## 利用可能なワークフロー

### `/release`
GitHub CLI プリコンパイル済み拡張機能のリリース手順

タグをプッシュすると GitHub Actions が自動でビルドとリリースを行います。

### `/create-issue`
gh-issuefs を使って新しい Issue を作成する手順

1. `sample` コマンドで Issue ディレクトリを生成
2. `issue.md` を編集
3. `push --new` で GitHub に新規 Issue 作成

## ワークフローの使い方

AI アシスタントに以下のように指示してください：

```
/release ワークフローを使って v1.0.2 をリリースして
```

```
/create-issue ワークフローを使って add_login_error_handling という Issue を作成して
```

各ワークフローの詳細は `.windsurf/workflows/` ディレクトリを参照してください。
