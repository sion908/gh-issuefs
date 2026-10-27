package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sion908/gh-issuefs/internal/app"
	"github.com/sion908/gh-issuefs/internal/ghcli"
)

var (
	globalApp *app.App
	globalCtx = context.Background()
)

type initCmd struct {
	Owner string `long:"owner" description:"Repository owner (detected from gh when omitted)"`
	Repo  string `long:"repo" description:"Repository name (detected from gh when omitted)"`
}

func (c *initCmd) Execute(_ []string) error {
	return globalApp.Init(globalCtx, c.Owner, c.Repo)
}

type configCmd struct {
	ListTemplates *listTemplatesCmd `command:"list-templates" description:"List files in .github/ISSUE_TEMPLATE/"`
	SetTemplate   *setTemplateCmd   `command:"set-template" description:"Set the default issue template" long-description:"Pass a filename for non-interactive use; omitting it opens a selection prompt. Example: gh issuefs config set-template bug_report.md"`
}

func (c *configCmd) Execute(_ []string) error {
	return globalApp.Config(globalCtx, false, "")
}

type listTemplatesCmd struct{}

func (c *listTemplatesCmd) Execute(_ []string) error {
	return globalApp.Config(globalCtx, true, "")
}

type setTemplateCmd struct {
	Positional struct {
		Name string `positional-arg-name:"name"`
	} `positional-args:"true"`
}

func (c *setTemplateCmd) Execute(_ []string) error {
	return globalApp.SetTemplate(globalCtx, c.Positional.Name)
}

type sampleCmd struct {
	Template   string `long:"template" description:"Filename in .github/ISSUE_TEMPLATE/ (overrides the configured default)"`
	Positional struct {
		Name string `positional-arg-name:"name" required:"true"`
	} `positional-args:"true"`
}

func (c *sampleCmd) Execute(_ []string) error {
	return globalApp.Sample(globalCtx, c.Positional.Name, c.Template)
}

type pullCmd struct {
	Positional struct {
		Numbers []string `positional-arg-name:"number"`
	} `positional-args:"true"`
	SaveRaw bool `long:"save-raw" description:"Also save the fetched issue data under .design/raw/"`
}

func (c *pullCmd) Execute(_ []string) error {
	return globalApp.Pull(globalCtx, app.PullOptions{SaveRaw: c.SaveRaw}, c.Positional.Numbers)
}

type pushCmd struct {
	Positional struct {
		Numbers []string `positional-arg-name:"number"`
	} `positional-args:"true"`
	New    bool `long:"new" description:"Create GitHub issues from ALL unnumbered issue directories"`
	DryRun bool `long:"dry-run" description:"Preview candidate writes without sending them to GitHub"`
}

func (c *pushCmd) Execute(_ []string) error {
	return globalApp.Push(globalCtx, app.PushOptions{New: c.New, DryRun: c.DryRun}, c.Positional.Numbers)
}

type syncDocsCmd struct {
	Positional struct {
		Files []string `positional-arg-name:"file"`
	} `positional-args:"true"`
	Remove bool `long:"remove" description:"Delete other files in the local .design/docs/ directory"`
}

func (c *syncDocsCmd) Execute(_ []string) error {
	return globalApp.SyncDocs(globalCtx, app.SyncDocsOptions{Remove: c.Remove}, c.Positional.Files)
}

type options struct {
	Init     initCmd     `command:"init" description:"Create local gh-issuefs configuration" long-description:"Create .design/config.toml and local issue/PR directories. Detects owner and repo through gh unless both are supplied. Example: gh issuefs init"`
	Config   configCmd   `command:"config" description:"Show or change the default template" long-description:"With no subcommand, print the configured default template. Use list-templates or set-template to inspect or change it. Example: gh issuefs config set-template bug_report.md"`
	Sample   sampleCmd   `command:"sample" description:"Create a local issue draft" long-description:"Write .design/issues/<name>/issue.md. This does not create a GitHub issue; push --new does that later. Example: gh issuefs sample add_login_error_handling"`
	Pull     pullCmd     `command:"pull" description:"Fetch issues and PRs into local files" long-description:"With numbers, fetch those issues or PRs. Without numbers, use sync.default_query from config (default: assignee:@me state:open). Writes local issue.md/pr.md and comments.json. Previously synced, locally modified issue.md is backed up before replacement. Example: gh issuefs pull 123"`
	Push     pushCmd     `command:"push" description:"Write local issue title and body to GitHub" long-description:"Without numbers, push every changed numbered issue directory. PRs are read-only. Use --dry-run to preview; --new creates GitHub issues from every unnumbered directory, not just one. Push does not check for newer remote edits. Example: gh issuefs push --dry-run 123"`
	SyncDocs syncDocsCmd `command:"sync-docs" description:"Copy root documentation into .design/docs/" long-description:"With no filenames, copy WORKFLOWS.md, SKILL_MANAGEMENT.md, README.md, and README.ja.md when present. --remove deletes other local files in .design/docs/. Example: gh issuefs sync-docs README.md"`
}

func main() {
	runner := ghcli.ExecRunner{}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	globalApp = app.New(root, runner, os.Stdout, os.Stderr)

	var opts options
	parser := flags.NewParser(&opts, flags.Default)
	parser.LongDescription = "Sync GitHub Issues and Pull Requests as local Markdown. Pull changes local files; push writes to GitHub. Run a command with --help for its defaults and side effects."
	parser.Find("config").SubcommandsOptional = true

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
