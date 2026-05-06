package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sion908/gh-issuefs/internal/issue"
)

// Sample generates a new issue template directory under .design/issues/<name>/.
// If template is non-empty, loads from .github/ISSUE_TEMPLATE/<template>.
func (a *App) Sample(_ context.Context, name, template string) error {
	cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	p := a.makePaths(cfg)

	dir := p.IssueDir(name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory already exists: %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	title := issue.TitleFromDirName(name)
	fm := issue.SampleFrontMatter(title)
	var body string

	templateToUse := template
	if templateToUse == "" {
		templateToUse = cfg.DefaultTemplate
	}

	if templateToUse != "" {
		templatePath := filepath.Join(a.Root, ".github", "ISSUE_TEMPLATE", templateToUse)
		data, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templateToUse, err)
		}
		body = string(data)
	} else {
		body = issue.SampleBody(title)
	}

	iss := issue.Issue{FrontMatter: fm, Body: body}

	mdPath := p.IssueMDPath(name)
	if err := issue.WriteFile(mdPath, iss); err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "Created %s\n", mdPath)
	return nil
}
