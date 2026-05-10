package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sion908/gh-issuefs/internal/config"
	"github.com/sion908/gh-issuefs/internal/ghcli"
	"github.com/sion908/gh-issuefs/internal/paths"
)

type App struct {
	Root   string
	Runner ghcli.Runner
	Out    io.Writer
	Err    io.Writer
}

type PullOptions struct {
	SaveRaw bool
}

type PushOptions struct {
	New    bool
	DryRun bool
}

func New(root string, runner ghcli.Runner, out, errOut io.Writer) *App {
	return &App{
		Root:   root,
		Runner: runner,
		Out:    out,
		Err:    errOut,
	}
}

// loadConfig loads the local config, falling back to global, then defaults.
func (a *App) loadConfig() (config.Config, error) {
	p := paths.New(a.Root, ".design", "issues", "pr")
	return config.LoadWithFallback(p.ConfigPath)
}

// makePaths returns a Paths built from config.
func (a *App) makePaths(cfg config.Config) paths.Paths {
	return paths.New(a.Root, cfg.Data.RootDir, cfg.Data.IssuesDir, cfg.Data.PrDir)
}

// detectRepo resolves "owner/repo" via gh CLI.
func (a *App) detectRepo(ctx context.Context) (string, error) {
	repo, err := ghcli.DetectRepo(ctx, a.Runner)
	if err != nil {
		return "", err
	}
	return repo, nil
}

// SetTemplate sets the default template interactively or directly.
func (a *App) SetTemplate(ctx context.Context, template string) error {
	cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	p := a.makePaths(cfg)

	if template == "" {
		// Interactive selection
		templates, err := a.getTemplateNames()
		if err != nil {
			return err
		}
		if len(templates) == 0 {
			fmt.Fprintln(a.Out, "No templates found in .github/ISSUE_TEMPLATE/")
			return nil
		}
		selected := ""
		prompt := &survey.Select{
			Message: "Select a template:",
			Options: templates,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return err
		}
		template = selected
	}

	cfg.DefaultTemplate = template
	if err := config.Save(p.ConfigPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Fprintf(a.Out, "Set default_template to %s\n", template)
	return nil
}

// Init initialises the .design directory and config.toml.
func (a *App) Init(ctx context.Context, owner, repo string) error {
	cfg := config.Default()
	p := a.makePaths(cfg)

	if _, err := os.Stat(p.ConfigPath); err == nil {
		return fmt.Errorf("config already exists at %s", p.ConfigPath)
	}

	if owner == "" || repo == "" {
		detected, err := a.detectRepo(ctx)
		if err != nil {
			return fmt.Errorf("unable to detect repository (use --owner/--repo): %w", err)
		}
		parts := splitSlash(detected)
		if len(parts) == 2 {
			if owner == "" {
				owner = parts[0]
			}
			if repo == "" {
				repo = parts[1]
			}
		}
	}

	if err := p.EnsureLayout(); err != nil {
		return err
	}
	if err := config.Save(p.ConfigPath, cfg); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Initialized gh-design for %s/%s in %s\n", owner, repo, p.DesignDir)
	return nil
}

func splitSlash(s string) []string {
	idx := -1
	for i, c := range s {
		if c == '/' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

func (a *App) getTemplateNames() ([]string, error) {
	templateDir := filepath.Join(a.Root, ".github", "ISSUE_TEMPLATE")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read templates: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
