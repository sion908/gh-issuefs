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
	Owner string `long:"owner" description:"Repository owner"`
	Repo  string `long:"repo" description:"Repository name"`
}

func (c *initCmd) Execute(_ []string) error {
	return globalApp.Init(globalCtx, c.Owner, c.Repo)
}

type configCmd struct {
	ListTemplates *listTemplatesCmd `command:"list-templates" description:"List available templates"`
	SetTemplate   *setTemplateCmd  `command:"set-template" description:"Set default template"`
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
	Template string `long:"template" description:"Template file name from .github/ISSUE_TEMPLATE/"`
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
	SaveRaw bool `long:"save-raw" description:"Save raw API responses"`
}

func (c *pullCmd) Execute(_ []string) error {
	return globalApp.Pull(globalCtx, app.PullOptions{SaveRaw: c.SaveRaw}, c.Positional.Numbers)
}

type pushCmd struct {
	Positional struct {
		Numbers []string `positional-arg-name:"number"`
	} `positional-args:"true"`
	New    bool `long:"new" description:"Create new issues from unnumbered directories"`
	DryRun bool `long:"dry-run" description:"Show what would be pushed without pushing"`
}

func (c *pushCmd) Execute(_ []string) error {
	return globalApp.Push(globalCtx, app.PushOptions{New: c.New, DryRun: c.DryRun}, c.Positional.Numbers)
}

type syncDocsCmd struct {
	Positional struct {
		Files []string `positional-arg-name:"file"`
	} `positional-args:"true"`
	Remove bool `long:"remove" description:"Remove files not in target list"`
}

func (c *syncDocsCmd) Execute(_ []string) error {
	return globalApp.SyncDocs(globalCtx, app.SyncDocsOptions{Remove: c.Remove}, c.Positional.Files)
}

type options struct {
	Init     initCmd     `command:"init"     description:"Initialize gh-design in the current repository"`
	Config   configCmd   `command:"config"   description:"Manage configuration"`
	Sample   sampleCmd   `command:"sample"   description:"Generate a new issue template"`
	Pull     pullCmd     `command:"pull"     description:"Pull issues from GitHub"`
	Push     pushCmd     `command:"push"     description:"Push issues to GitHub"`
	SyncDocs syncDocsCmd `command:"sync-docs" description:"Sync project documentation to .design/docs/"`
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

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
