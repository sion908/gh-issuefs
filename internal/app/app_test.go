package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sion908/gh-issuefs/internal/issue"
)

// mockRunner is a local mock for testing
type mockRunner struct {
	Output string
	Err    error
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return m.Output, m.Err
}

// seqRunner returns different outputs per call in sequence.
type seqRunner struct {
	responses []mockResponse
	idx       int
}

type mockResponse struct {
	Output string
	Err    error
}

func (s *seqRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if s.idx >= len(s.responses) {
		return "", nil
	}
	r := s.responses[s.idx]
	s.idx++
	return r.Output, r.Err
}

func TestNew(t *testing.T) {
	runner := &mockRunner{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	app := New("/root", runner, out, errOut)

	if app.Root != "/root" {
		t.Errorf("expected Root /root, got %s", app.Root)
	}
	if app.Runner != runner {
		t.Error("runner not set correctly")
	}
	if app.Out != out {
		t.Error("Out not set correctly")
	}
	if app.Err != errOut {
		t.Error("Err not set correctly")
	}
}

func TestSplitSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"owner/repo", []string{"owner", "repo"}},
		{"org/project", []string{"org", "project"}},
		{"single", []string{"single"}},
		{"", []string{""}},
		{"a/b/c", []string{"a", "b/c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitSlash(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d parts, got %d", len(tt.expected), len(result))
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("part %d: expected %s, got %s", i, tt.expected[i], part)
				}
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{9999, "9999"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoteToIssue(t *testing.T) {
	ri := issue.RemoteIssue{
		Number:    123,
		Title:     "Test Issue",
		Body:      "Test body",
		State:     "open",
		URL:       "https://github.com/owner/repo/issues/123",
		Labels:    []string{"bug", "enhancement"},
		Assignees: []string{"user1", "user2"},
		Milestone: stringPtr("v1.0"),
		ProjectsV2: []issue.ProjectV2Item{
			{Name: "Project A", Status: "In Progress"},
		},
	}

	iss := remoteToIssue(ri)

	if iss.FrontMatter.Number != 123 {
		t.Errorf("expected number 123, got %d", iss.FrontMatter.Number)
	}
	if iss.FrontMatter.Title != "Test Issue" {
		t.Errorf("expected title Test Issue, got %s", iss.FrontMatter.Title)
	}
	if iss.FrontMatter.State != "open" {
		t.Errorf("expected state open, got %s", iss.FrontMatter.State)
	}
	if len(iss.FrontMatter.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(iss.FrontMatter.Labels))
	}
	if len(iss.FrontMatter.Assignees) != 2 {
		t.Errorf("expected 2 assignees, got %d", len(iss.FrontMatter.Assignees))
	}
	if iss.FrontMatter.Milestone == nil || *iss.FrontMatter.Milestone != "v1.0" {
		t.Error("milestone not set correctly")
	}
	if len(iss.FrontMatter.ProjectsV2) != 1 {
		t.Errorf("expected 1 project, got %d", len(iss.FrontMatter.ProjectsV2))
	}
	if iss.Body != "Test body" {
		t.Errorf("expected body Test body, got %s", iss.Body)
	}
}

func TestEvacuatePath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("first backup", func(t *testing.T) {
		mdPath := filepath.Join(tmpDir, "issue.md")
		result := evacuatePath(mdPath)
		expected := filepath.Join(tmpDir, "issue.local.md")
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("second backup", func(t *testing.T) {
		mdPath := filepath.Join(tmpDir, "issue.md")
		// Create first backup
		firstBackup := filepath.Join(tmpDir, "issue.local.md")
		if err := os.WriteFile(firstBackup, []byte("backup"), 0o644); err != nil {
			t.Fatalf("failed to create first backup: %v", err)
		}

		result := evacuatePath(mdPath)
		expected := filepath.Join(tmpDir, "issue.local.1.md")
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("third backup", func(t *testing.T) {
		mdPath := filepath.Join(tmpDir, "issue.md")
		// Create first and second backups
		for i := 0; i < 2; i++ {
			backup := filepath.Join(tmpDir, "issue.local.md")
			if i == 1 {
				backup = filepath.Join(tmpDir, "issue.local.1.md")
			}
			if err := os.WriteFile(backup, []byte("backup"), 0o644); err != nil {
				t.Fatalf("failed to create backup: %v", err)
			}
		}

		result := evacuatePath(mdPath)
		expected := filepath.Join(tmpDir, "issue.local.2.md")
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})
}

func TestPrintDiff(t *testing.T) {
	tests := []struct {
		name     string
		oldVal   string
		newVal   string
		field    string
		contains string
	}{
		{
			name:     "changed",
			oldVal:   "old",
			newVal:   "new",
			field:    "title",
			contains: "title: \"old\" → \"new\"",
		},
		{
			name:     "unchanged",
			oldVal:   "same",
			newVal:   "same",
			field:    "title",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			printDiff(buf, tt.oldVal, tt.newVal, tt.field)
			output := buf.String()

			if tt.contains == "" {
				if output != "" {
					t.Errorf("expected empty output, got %s", output)
				}
			} else {
				if !strings.Contains(output, tt.contains) {
					t.Errorf("expected output to contain %s, got %s", tt.contains, output)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	app := New(tmpDir, &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})

	// Test with no config (should return default)
	cfg, err := app.loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Data.RootDir != ".design" {
		t.Errorf("expected default RootDir, got %s", cfg.Data.RootDir)
	}

	// Test with config file
	p := filepath.Join(tmpDir, ".design", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	cfg.DefaultTemplate = "test.md"
	if err := app.makePaths(cfg).EnsureLayout(); err != nil {
		t.Fatalf("failed to ensure layout: %v", err)
	}

	loaded, err := app.loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still return default since we didn't save the config
	if loaded.Data.RootDir != ".design" {
		t.Errorf("expected default RootDir, got %s", loaded.Data.RootDir)
	}
}

func TestMakePaths(t *testing.T) {
	app := New("/root", &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})
	cfg, err := app.loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Data.RootDir != ".design" {
		t.Fatalf("unexpected config: %v", cfg)
	}

	p := app.makePaths(cfg)
	if p.Root != "/root" {
		t.Errorf("expected Root /root, got %s", p.Root)
	}
	if p.DesignDir != "/root/.design" {
		t.Errorf("expected DesignDir /root/.design, got %s", p.DesignDir)
	}
}

func TestDetectRepo(t *testing.T) {
	ctx := context.Background()
	app := New("/root", &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})

	t.Run("success", func(t *testing.T) {
		runner := &mockRunner{Output: "owner/repo\n"}
		app.Runner = runner

		repo, err := app.detectRepo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo != "owner/repo" {
			t.Errorf("expected owner/repo, got %s", repo)
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &mockRunner{Err: errors.New("mock error")}
		app.Runner = runner

		_, err := app.detectRepo(ctx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetTemplateNames(t *testing.T) {
	tmpDir := t.TempDir()
	app := New(tmpDir, &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})

	t.Run("no templates dir", func(t *testing.T) {
		names, err := app.getTemplateNames()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if names != nil {
			t.Errorf("expected nil, got %v", names)
		}
	})

	t.Run("with templates", func(t *testing.T) {
		templateDir := filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE")
		if err := os.MkdirAll(templateDir, 0o755); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}

		templates := []string{"default.md", "bug_report.md", "feature.md"}
		for _, tmpl := range templates {
			if err := os.WriteFile(filepath.Join(templateDir, tmpl), []byte("test"), 0o644); err != nil {
				t.Fatalf("failed to create template: %v", err)
			}
		}

		// Create a subdirectory (should be ignored)
		if err := os.Mkdir(filepath.Join(templateDir, "subdir"), 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		names, err := app.getTemplateNames()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 3 {
			t.Errorf("expected 3 templates, got %d", len(names))
		}
	})
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	t.Run("success with owner/repo", func(t *testing.T) {
		app := New(tmpDir, &mockRunner{}, out, errOut)

		err := app.Init(ctx, "owner", "repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check config was created
		configPath := filepath.Join(tmpDir, ".design", "config.toml")
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("config not created: %v", err)
		}

		// Check output
		if !strings.Contains(out.String(), "Initialized gh-design for owner/repo") {
			t.Errorf("expected success message, got %s", out.String())
		}
	})

	t.Run("config already exists", func(t *testing.T) {
		app := New(tmpDir, &mockRunner{}, out, errOut)

		// Create config first
		configPath := filepath.Join(tmpDir, ".design", "config.toml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create config: %v", err)
		}

		err := app.Init(ctx, "owner", "repo")
		if err == nil {
			t.Error("expected error for existing config, got nil")
		}
	})

	t.Run("detect repo", func(t *testing.T) {
		// Use a different tmpDir to avoid conflict with previous test
		detectTmpDir := t.TempDir()
		app := New(detectTmpDir, &mockRunner{}, out, errOut)
		app.Runner = &mockRunner{Output: "detected/detected-repo\n"}

		err := app.Init(ctx, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "detected/detected-repo") {
			t.Errorf("expected detected repo in output, got %s", out.String())
		}
	})
}

func TestConfig(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := New(tmpDir, &mockRunner{}, out, errOut)

	t.Run("show config", func(t *testing.T) {
		err := app.Config(ctx, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "default_template:") {
			t.Errorf("expected default_template in output, got %s", out.String())
		}
	})

	t.Run("list templates", func(t *testing.T) {
		templateDir := filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE")
		if err := os.MkdirAll(templateDir, 0o755); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(templateDir, "test.md"), []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}

		out.Reset()
		err := app.Config(ctx, true, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "test.md") {
			t.Errorf("expected test.md in output, got %s", out.String())
		}
	})
}

func TestSample(t *testing.T) {
	ctx := context.Background()

	t.Run("creates issue dir and file", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		app := New(tmpDir, &mockRunner{}, out, &bytes.Buffer{})

		err := app.Sample(ctx, "add_feature", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mdPath := filepath.Join(tmpDir, ".design", "issues", "add_feature", "issue.md")
		if _, err := os.Stat(mdPath); err != nil {
			t.Errorf("issue.md not created: %v", err)
		}
		if !strings.Contains(out.String(), "Created") {
			t.Errorf("expected Created in output, got %s", out.String())
		}
	})

	t.Run("error when dir already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		app := New(tmpDir, &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})

		// 1回目は成功
		if err := app.Sample(ctx, "add_feature", ""); err != nil {
			t.Fatalf("first Sample failed: %v", err)
		}
		// 2回目は既存ディレクトリエラー
		if err := app.Sample(ctx, "add_feature", ""); err == nil {
			t.Error("expected error for existing directory, got nil")
		}
	})

	t.Run("with template", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		app := New(tmpDir, &mockRunner{}, out, &bytes.Buffer{})

		// テンプレートファイルを用意
		templateDir := filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE")
		if err := os.MkdirAll(templateDir, 0o755); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}
		templateContent := "## Bug Report\n\nDescribe the bug here."
		if err := os.WriteFile(filepath.Join(templateDir, "bug_report.md"), []byte(templateContent), 0o644); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}

		err := app.Sample(ctx, "fix_bug", "bug_report.md")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mdPath := filepath.Join(tmpDir, ".design", "issues", "fix_bug", "issue.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("failed to read issue.md: %v", err)
		}
		if !strings.Contains(string(data), "Bug Report") {
			t.Errorf("expected template content in issue.md, got %s", string(data))
		}
	})

	t.Run("template not found returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		app := New(tmpDir, &mockRunner{}, &bytes.Buffer{}, &bytes.Buffer{})

		err := app.Sample(ctx, "new_issue", "nonexistent.md")
		if err == nil {
			t.Error("expected error for missing template, got nil")
		}
	})
}

// graphQL レスポンスのヘルパー
func graphQLIssueResp(number int, title, body, state string) string {
	return `{"data":{"repository":{"issue":{` +
		`"number":` + itoa(number) + `,` +
		`"title":"` + title + `",` +
		`"body":"` + body + `",` +
		`"state":"` + strings.ToUpper(state) + `",` +
		`"url":"https://github.com/owner/repo/issues/` + itoa(number) + `",` +
		`"__typename":"Issue",` +
		`"updatedAt":"2024-01-01T00:00:00Z",` +
		`"labels":{"nodes":[]},` +
		`"assignees":{"nodes":[]},` +
		`"milestone":null,` +
		`"projectItems":{"nodes":[]}}}}}`
}

func TestPull(t *testing.T) {
	ctx := context.Background()

	t.Run("pulls single issue by number", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		runner := &seqRunner{responses: []mockResponse{
			{Output: "owner/repo\n"},                                      // DetectRepo
			{Output: graphQLIssueResp(42, "Test Issue", "Body", "open")}, // GetIssue #42
			{Output: "[]"},                                                // GetComments
		}}
		app := New(tmpDir, runner, out, errOut)

		err := app.Pull(ctx, PullOptions{}, []string{"42"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mdPath := filepath.Join(tmpDir, ".design", "issues", "42", "issue.md")
		if _, err := os.Stat(mdPath); err != nil {
			t.Errorf("issue.md not created: %v", err)
		}
		if !strings.Contains(out.String(), "pulled #42") {
			t.Errorf("expected pulled message, got %s", out.String())
		}
	})

	t.Run("backs up local changes before overwrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}

		// 先に issue.md と .meta.json を作成してローカル変更をシミュレート
		issueDir := filepath.Join(tmpDir, ".design", "issues", "42")
		if err := os.MkdirAll(issueDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		mdPath := filepath.Join(issueDir, "issue.md")
		// 元のコンテンツを書き込む
		origContent := []byte("---\nnumber: 42\ntitle: Original\n---\nOriginal body\n")
		if err := os.WriteFile(mdPath, origContent, 0o644); err != nil {
			t.Fatalf("failed to write issue.md: %v", err)
		}
		// .meta.json: 元のハッシュを記録（ここでは別のハッシュにして「変更あり」を模擬）
		meta := issue.Meta{
			Number:        42,
			Title:         "Original",
			IssueMDSHA256: "differenthash",
		}
		metaPath := filepath.Join(issueDir, ".meta.json")
		if err := issue.SaveMeta(metaPath, meta); err != nil {
			t.Fatalf("failed to save meta: %v", err)
		}

		runner := &seqRunner{responses: []mockResponse{
			{Output: "owner/repo\n"},
			{Output: graphQLIssueResp(42, "Updated Issue", "New body", "open")},
			{Output: "[]"},
		}}
		app := New(tmpDir, runner, out, &bytes.Buffer{})

		err := app.Pull(ctx, PullOptions{}, []string{"42"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// バックアップが作成されているか確認
		backupPath := filepath.Join(issueDir, "issue.local.md")
		if _, err := os.Stat(backupPath); err != nil {
			t.Errorf("backup not created: %v", err)
		}
		if !strings.Contains(out.String(), "backed up") {
			t.Errorf("expected backup message, got %s", out.String())
		}
	})

	t.Run("saves raw when SaveRaw is true", func(t *testing.T) {
		tmpDir := t.TempDir()

		runner := &seqRunner{responses: []mockResponse{
			{Output: "owner/repo\n"},
			{Output: graphQLIssueResp(5, "Raw Issue", "Body", "open")},
			{Output: "[]"},
		}}
		app := New(tmpDir, runner, &bytes.Buffer{}, &bytes.Buffer{})

		err := app.Pull(ctx, PullOptions{SaveRaw: true}, []string{"5"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		rawPath := filepath.Join(tmpDir, ".design", "raw", "issues", "5.json")
		if _, err := os.Stat(rawPath); err != nil {
			t.Errorf("raw file not saved: %v", err)
		}
	})

	t.Run("error on invalid issue number", func(t *testing.T) {
		tmpDir := t.TempDir()
		app := New(tmpDir, &mockRunner{Output: "owner/repo\n"}, &bytes.Buffer{}, &bytes.Buffer{})

		err := app.Pull(ctx, PullOptions{}, []string{"abc"})
		if err == nil {
			t.Error("expected error for invalid issue number, got nil")
		}
	})

	t.Run("error when DetectRepo fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		app := New(tmpDir, &mockRunner{Err: errors.New("no repo")}, &bytes.Buffer{}, &bytes.Buffer{})

		err := app.Pull(ctx, PullOptions{}, []string{"1"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestPush(t *testing.T) {
	ctx := context.Background()

	// issue.md と .meta.json を作成するヘルパー
	setupIssueDir := func(t *testing.T, issuesDir string, number int, title, body string) (dirName string) {
		t.Helper()
		dirName = itoa(number) + "_" + strings.ReplaceAll(strings.ToLower(title), " ", "_")
		dir := filepath.Join(issuesDir, dirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		iss := issue.Issue{
			FrontMatter: issue.FrontMatter{Number: number, Title: title, State: "open"},
			Body:        body,
		}
		data, err := issue.Render(iss)
		if err != nil {
			t.Fatalf("failed to render issue: %v", err)
		}
		mdPath := filepath.Join(dir, "issue.md")
		if err := os.WriteFile(mdPath, data, 0o644); err != nil {
			t.Fatalf("failed to write issue.md: %v", err)
		}
		meta := issue.Meta{
			Number:        number,
			Title:         title,
			IssueMDSHA256: issue.SHA256Hex(data),
		}
		if err := issue.SaveMeta(filepath.Join(dir, ".meta.json"), meta); err != nil {
			t.Fatalf("failed to save meta: %v", err)
		}
		return dirName
	}

	t.Run("nothing to push when no changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		issuesDir := filepath.Join(tmpDir, ".design", "issues")
		if err := os.MkdirAll(issuesDir, 0o755); err != nil {
			t.Fatalf("failed to create issues dir: %v", err)
		}
		setupIssueDir(t, issuesDir, 10, "No Change", "body")

		app := New(tmpDir, &mockRunner{Output: "owner/repo\n"}, out, &bytes.Buffer{})
		err := app.Push(ctx, PushOptions{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "nothing to push") {
			t.Errorf("expected 'nothing to push', got %s", out.String())
		}
	})

	t.Run("pushes changed issue", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		issuesDir := filepath.Join(tmpDir, ".design", "issues")
		if err := os.MkdirAll(issuesDir, 0o755); err != nil {
			t.Fatalf("failed to create issues dir: %v", err)
		}
		dirName := setupIssueDir(t, issuesDir, 20, "Original Title", "body")

		// issue.md を書き換えてハッシュを変化させる
		mdPath := filepath.Join(issuesDir, dirName, "issue.md")
		iss := issue.Issue{
			FrontMatter: issue.FrontMatter{Number: 20, Title: "Updated Title", State: "open"},
			Body:        "updated body",
		}
		data, _ := issue.Render(iss)
		if err := os.WriteFile(mdPath, data, 0o644); err != nil {
			t.Fatalf("failed to overwrite issue.md: %v", err)
		}

		runner := &seqRunner{responses: []mockResponse{
			{Output: "owner/repo\n"}, // DetectRepo
			{Output: ""},             // EditIssue
		}}
		app := New(tmpDir, runner, out, &bytes.Buffer{})

		err := app.Push(ctx, PushOptions{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "pushed #20") {
			t.Errorf("expected pushed message, got %s", out.String())
		}
	})

	t.Run("dry-run does not push", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		issuesDir := filepath.Join(tmpDir, ".design", "issues")
		if err := os.MkdirAll(issuesDir, 0o755); err != nil {
			t.Fatalf("failed to create issues dir: %v", err)
		}
		dirName := setupIssueDir(t, issuesDir, 30, "Dry Run Issue", "body")

		// issue.md を変更
		mdPath := filepath.Join(issuesDir, dirName, "issue.md")
		iss := issue.Issue{
			FrontMatter: issue.FrontMatter{Number: 30, Title: "Dry Run Updated", State: "open"},
			Body:        "changed body",
		}
		data, _ := issue.Render(iss)
		if err := os.WriteFile(mdPath, data, 0o644); err != nil {
			t.Fatalf("failed to write issue.md: %v", err)
		}

		app := New(tmpDir, &mockRunner{Output: "owner/repo\n"}, out, &bytes.Buffer{})
		err := app.Push(ctx, PushOptions{DryRun: true}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "[dry-run]") {
			t.Errorf("expected dry-run message, got %s", out.String())
		}
	})

	t.Run("push --new creates new issue and renames dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := &bytes.Buffer{}
		issuesDir := filepath.Join(tmpDir, ".design", "issues")
		if err := os.MkdirAll(issuesDir, 0o755); err != nil {
			t.Fatalf("failed to create issues dir: %v", err)
		}

		// 番号なしディレクトリに issue.md を作成
		newDir := filepath.Join(issuesDir, "new_feature")
		if err := os.MkdirAll(newDir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		iss := issue.Issue{
			FrontMatter: issue.FrontMatter{Title: "New Feature"},
			Body:        "feature body",
		}
		data, _ := issue.Render(iss)
		if err := os.WriteFile(filepath.Join(newDir, "issue.md"), data, 0o644); err != nil {
			t.Fatalf("failed to write issue.md: %v", err)
		}

		runner := &seqRunner{responses: []mockResponse{
			{Output: "owner/repo\n"},                                // DetectRepo
			{Output: "https://github.com/owner/repo/issues/99\n"},  // CreateIssue
		}}
		app := New(tmpDir, runner, out, &bytes.Buffer{})

		err := app.Push(ctx, PushOptions{New: true}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "created #99") {
			t.Errorf("expected created message, got %s", out.String())
		}
		// ディレクトリがリネームされているか確認
		renamedDir := filepath.Join(issuesDir, "99_new_feature")
		if _, err := os.Stat(renamedDir); err != nil {
			t.Errorf("renamed dir not found: %v", err)
		}
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

