package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sion908/gh-issuefs/internal/ghcli"
	"github.com/sion908/gh-issuefs/internal/issue"
	"github.com/sion908/gh-issuefs/internal/paths"
)

// Pull syncs GitHub Issues to local .design/issues/.
// If numbers is empty, the default query from config is used.
func (a *App) Pull(ctx context.Context, opts PullOptions, numbers []string) error {
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

	if err := p.EnsureLayout(); err != nil {
		return err
	}

	var remoteIssues []issue.RemoteIssue

	if len(numbers) > 0 {
		for _, numStr := range numbers {
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return fmt.Errorf("invalid issue number %q: %w", numStr, err)
			}
			ri, err := client.GetIssue(ctx, n)
			if err != nil {
				return fmt.Errorf("failed to fetch issue #%d: %w", n, err)
			}
			if ri.IsPullRequest {
				fmt.Fprintf(a.Err, "skipping #%d (pull request)\n", n)
				continue
			}
			remoteIssues = append(remoteIssues, ri)
		}
	} else {
		query := cfg.Sync.DefaultQuery
		all, err := client.ListIssues(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}
		for _, ri := range all {
			if cfg.Sync.ExcludePullRequests && ri.IsPullRequest {
				continue
			}
			remoteIssues = append(remoteIssues, ri)
		}
	}

	for _, ri := range remoteIssues {
		if err := a.syncIssue(ctx, p, client, ri, opts); err != nil {
			fmt.Fprintf(a.Err, "error syncing #%d: %v\n", ri.Number, err)
		}
	}
	return nil
}

// syncIssue writes issue.md, comments.json, .meta.json for a single remote issue.
func (a *App) syncIssue(ctx context.Context, p paths.Paths, client *ghcli.Client, ri issue.RemoteIssue, opts PullOptions) error {
	dirName := findOrCreateDirName(p, ri.Number)
	issueDir := p.IssueDir(dirName)

	if err := os.MkdirAll(issueDir, 0o755); err != nil {
		return err
	}

	mdPath := p.IssueMDPath(dirName)
	metaPath := p.MetaPath(dirName)

	// --- Local change protection ---
	if _, err := os.Stat(mdPath); err == nil {
		existingData, err := os.ReadFile(mdPath)
		if err == nil {
			existingHash := issue.SHA256Hex(existingData)
			meta, metaErr := issue.LoadMeta(metaPath)
			if metaErr == nil && meta.IssueMDSHA256 != "" && existingHash != meta.IssueMDSHA256 {
				// Local modification detected: back up
				backup := evacuatePath(mdPath)
				if err := os.Rename(mdPath, backup); err != nil {
					return fmt.Errorf("failed to back up modified issue.md: %w", err)
				}
				fmt.Fprintf(a.Out, "#%d: local changes backed up to %s\n", ri.Number, filepath.Base(backup))
			}
		}
	}

	// --- Write issue.md ---
	iss := remoteToIssue(ri)
	newData, err := issue.Render(iss)
	if err != nil {
		return err
	}
	if err := os.WriteFile(mdPath, newData, 0o644); err != nil {
		return err
	}

	// --- Write comments.json ---
	if err := a.writeComments(ctx, p, dirName, client, ri.Number); err != nil {
		fmt.Fprintf(a.Err, "#%d: failed to fetch comments: %v\n", ri.Number, err)
	}

	// --- Write .meta.json ---
	meta := issue.Meta{
		Number:           ri.Number,
		ID:               ri.ID,
		Title:            ri.Title,
		State:            ri.State,
		URL:              ri.URL,
		IsPullRequest:    ri.IsPullRequest,
		UpdatedAt:        ri.UpdatedAt,
		LastSyncedAt:     time.Now().UTC(),
		IssueMDSHA256:    issue.SHA256Hex(newData),
		RemoteBodySHA256: issue.SHA256Hex([]byte(ri.Body)),
	}
	if err := issue.SaveMeta(metaPath, meta); err != nil {
		return err
	}

	// --- Raw save ---
	if opts.SaveRaw {
		if err := a.saveRaw(p, ri); err != nil {
			fmt.Fprintf(a.Err, "#%d: failed to save raw: %v\n", ri.Number, err)
		}
	}

	fmt.Fprintf(a.Out, "pulled #%d %s\n", ri.Number, ri.Title)
	return nil
}

func (a *App) writeComments(ctx context.Context, p paths.Paths, dirName string, client *ghcli.Client, number int) error {
	comments, err := client.GetComments(ctx, number)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.CommentsPath(dirName), data, 0o644)
}

func (a *App) saveRaw(p paths.Paths, ri issue.RemoteIssue) error {
	path := p.RawIssuePath(ri.Number)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ri, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// findOrCreateDirName finds an existing issue directory by number prefix,
// or returns a new directory name "<number>" if none exists.
func findOrCreateDirName(p paths.Paths, number int) string {
	dirs, _ := p.ListIssueDirs()
	for _, d := range dirs {
		if paths.IssueDirNumber(d) == number {
			return d
		}
	}
	return paths.IssueDirName(number, "")
}

// remoteToIssue converts a RemoteIssue to an Issue for writing issue.md.
func remoteToIssue(ri issue.RemoteIssue) issue.Issue {
	fm := issue.FrontMatter{
		Number:     ri.Number,
		Title:      ri.Title,
		State:      ri.State,
		Labels:     ri.Labels,
		Assignees:  ri.Assignees,
		Milestone:  ri.Milestone,
		ProjectsV2: ri.ProjectsV2,
	}
	return issue.Issue{FrontMatter: fm, Body: ri.Body}
}

// evacuatePath returns a backup path for an issue.md file.
// e.g. "issue.md" -> "issue.local.md", then "issue.local.1.md", etc.
func evacuatePath(mdPath string) string {
	dir := filepath.Dir(mdPath)
	base := "issue.local.md"
	candidate := filepath.Join(dir, base)
	if _, err := os.Stat(candidate); err != nil {
		return candidate
	}
	for i := 1; ; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("issue.local.%d.md", i))
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}
