package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sion908/gh-issuefs/internal/ghcli"
	"github.com/sion908/gh-issuefs/internal/issue"
	"github.com/sion908/gh-issuefs/internal/paths"
)

// Push pushes local issue.md changes to GitHub.
// If opts.New is set, creates new issues from unnumbered directories.
// If numbers is empty and opts.New is false, pushes all numbered dirs with changes.
func (a *App) Push(ctx context.Context, opts PushOptions, numbers []string) error {
	cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	p := a.makePaths(cfg)

	repo, err := a.detectRepo(ctx)
	if err != nil {
		return err
	}
	client := ghcli.NewClient(a.Runner, repo)

	if opts.New {
		return a.pushNew(ctx, p, client, opts)
	}

	dirs, err := p.ListIssueDirs()
	if err != nil {
		return err
	}

	// Determine target directories
	var targets []string
	if len(numbers) > 0 {
		numSet := make(map[string]bool)
		for _, n := range numbers {
			numSet[n] = true
		}
		for _, d := range dirs {
			n := paths.IssueDirNumber(d)
			if n > 0 && numSet[itoa(n)] {
				targets = append(targets, d)
			}
		}
	} else {
		for _, d := range dirs {
			if paths.IssueDirNumber(d) > 0 {
				targets = append(targets, d)
			}
		}
	}

	pushed := 0
	for _, dirName := range targets {
		changed, err := a.pushIssue(ctx, p, client, dirName, opts)
		if err != nil {
			fmt.Fprintf(a.Err, "error pushing %s: %v\n", dirName, err)
			continue
		}
		if changed {
			pushed++
		}
	}
	if pushed == 0 {
		fmt.Fprintln(a.Out, "nothing to push")
	}
	return nil
}

// pushIssue pushes a single numbered issue directory. Returns true if a push was performed.
func (a *App) pushIssue(ctx context.Context, p paths.Paths, client *ghcli.Client, dirName string, opts PushOptions) (bool, error) {
	mdPath := p.IssueMDPath(dirName)
	metaPath := p.MetaPath(dirName)

	data, err := os.ReadFile(mdPath)
	if err != nil {
		return false, fmt.Errorf("cannot read %s: %w", mdPath, err)
	}

	currentHash := issue.SHA256Hex(data)

	meta, err := issue.LoadMeta(metaPath)
	if err != nil {
		return false, fmt.Errorf("cannot read .meta.json for %s (run pull first): %w", dirName, err)
	}

	// No local changes
	if currentHash == meta.IssueMDSHA256 {
		return false, nil
	}

	iss, err := issue.ParseFile(mdPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s: %w", mdPath, err)
	}

	number := meta.Number

	// Block push for pull requests
	if meta.IsPullRequest {
		return false, fmt.Errorf("pull requests cannot be pushed (read-only)")
	}

	// Show diff
	fmt.Fprintf(a.Out, "--- #%d %s ---\n", number, meta.Title)
	printDiff(a.Out, meta.Title, iss.FrontMatter.Title, "title")
	if iss.Body != "" {
		fmt.Fprintf(a.Out, "  body: changed\n")
	}

	if opts.DryRun {
		fmt.Fprintf(a.Out, "[dry-run] would push #%d\n", number)
		return false, nil
	}

	if err := client.EditIssue(ctx, number, iss.FrontMatter.Title, iss.Body); err != nil {
		return false, fmt.Errorf("failed to push #%d: %w", number, err)
	}

	// Update .meta.json with new hash
	meta.Title = iss.FrontMatter.Title
	meta.IssueMDSHA256 = currentHash
	meta.RemoteBodySHA256 = issue.SHA256Hex([]byte(iss.Body))
	if err := issue.SaveMeta(metaPath, meta); err != nil {
		return true, err
	}

	fmt.Fprintf(a.Out, "pushed #%d %s\n", number, iss.FrontMatter.Title)
	return true, nil
}

// pushNew creates new GitHub Issues from unnumbered directories.
func (a *App) pushNew(ctx context.Context, p paths.Paths, client *ghcli.Client, opts PushOptions) error {
	dirs, err := p.ListIssueDirs()
	if err != nil {
		return err
	}

	for _, dirName := range dirs {
		if paths.IssueDirNumber(dirName) != 0 {
			continue
		}
		mdPath := p.IssueMDPath(dirName)
		if _, err := os.Stat(mdPath); err != nil {
			continue
		}

		iss, err := issue.ParseFile(mdPath)
		if err != nil {
			fmt.Fprintf(a.Err, "skipping %s: %v\n", dirName, err)
			continue
		}
		// Skip if already has a number in frontmatter
		if iss.FrontMatter.Number > 0 {
			continue
		}

		fmt.Fprintf(a.Out, "creating issue: %s\n", iss.FrontMatter.Title)

		if opts.DryRun {
			fmt.Fprintf(a.Out, "[dry-run] would create issue from %s\n", dirName)
			continue
		}

		number, nodeID, err := client.CreateIssue(ctx, iss.FrontMatter.Title, iss.Body)
		if err != nil {
			fmt.Fprintf(a.Err, "failed to create issue from %s: %v\n", dirName, err)
			continue
		}

		// Rename directory to include issue number
		newDirName := paths.IssueDirName(number, dirName)
		newDir := p.IssueDir(newDirName)
		oldDir := p.IssueDir(dirName)
		if err := os.Rename(oldDir, newDir); err != nil {
			fmt.Fprintf(a.Err, "created #%d but failed to rename directory: %v\n", number, err)
			continue
		}

		// Update issue.md frontmatter with number
		iss.FrontMatter.Number = number
		iss.FrontMatter.State = "open"
		newData, err := issue.Render(iss)
		if err == nil {
			newMdPath := p.IssueMDPath(newDirName)
			_ = os.WriteFile(newMdPath, newData, 0o644)
		}

		// Write .meta.json
		meta := issue.Meta{
			Number:        number,
			ID:            nodeID,
			Title:         iss.FrontMatter.Title,
			State:         "open",
			IsPullRequest: false,
		}
		if len(newData) > 0 {
			meta.IssueMDSHA256 = issue.SHA256Hex(newData)
		}
		meta.RemoteBodySHA256 = issue.SHA256Hex([]byte(iss.Body))
		_ = issue.SaveMeta(p.MetaPath(newDirName), meta)

		fmt.Fprintf(a.Out, "created #%d → %s\n", number, newDirName)
	}
	return nil
}

func printDiff(out interface{ Write([]byte) (int, error) }, oldVal, newVal, field string) {
	if oldVal != newVal {
		fmt.Fprintf(out, "  %s: %q → %q\n", field, oldVal, newVal)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// unused import guard
var _ = strings.TrimSpace
