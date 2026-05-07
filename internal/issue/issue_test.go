package issue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		check   func(Issue) bool
	}{
		{
			name: "valid frontmatter",
			data: `---
number: 123
title: Test Issue
state: open
labels:
  - bug
assignees:
  - user1
---

This is the body`,
			wantErr: false,
			check: func(iss Issue) bool {
				return iss.FrontMatter.Number == 123 &&
					iss.FrontMatter.Title == "Test Issue" &&
					iss.FrontMatter.State == "open" &&
					len(iss.FrontMatter.Labels) == 1 &&
					iss.FrontMatter.Labels[0] == "bug" &&
					len(iss.FrontMatter.Assignees) == 1 &&
					iss.FrontMatter.Assignees[0] == "user1" &&
					iss.Body == "This is the body"
			},
		},
		{
			name: "no frontmatter",
			data: `Just plain markdown without frontmatter`,
			wantErr: false,
			check: func(iss Issue) bool {
				return iss.FrontMatter.Title == "" &&
					iss.Body == "Just plain markdown without frontmatter"
			},
		},
		{
			name:    "unclosed frontmatter",
			data:    "---\nnumber: 123\n\nbody without closing",
			wantErr: true,
			check:   nil,
		},
		{
			name: "empty body",
			data: `---
title: Empty Body
---
`,
			wantErr: false,
			check: func(iss Issue) bool {
				return iss.FrontMatter.Title == "Empty Body" && iss.Body == ""
			},
		},
		{
			name: "with projects_v2",
			data: `---
title: Test
projects_v2:
  - name: Project A
    status: In Progress
---
Body`,
			wantErr: false,
			check: func(iss Issue) bool {
				return len(iss.FrontMatter.ProjectsV2) == 1 &&
					iss.FrontMatter.ProjectsV2[0].Name == "Project A" &&
					iss.FrontMatter.ProjectsV2[0].Status == "In Progress"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iss, err := Parse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(iss) {
				t.Error("Parse() result check failed")
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "issue.md")

	testData := `---
number: 123
title: Test
---
Body`
	if err := os.WriteFile(testFile, []byte(testData), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	iss, err := ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile() failed: %v", err)
	}
	if iss.FrontMatter.Number != 123 {
		t.Errorf("expected number 123, got %d", iss.FrontMatter.Number)
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		iss     Issue
		wantErr bool
		check   func([]byte) bool
	}{
		{
			name: "basic render",
			iss: Issue{
				FrontMatter: FrontMatter{
					Number: 123,
					Title:  "Test Issue",
					State:  "open",
				},
				Body: "Test body",
			},
			wantErr: false,
			check: func(data []byte) bool {
				s := string(data)
				return contains(s, "---") &&
					contains(s, "number: 123") &&
					contains(s, "title: Test Issue") &&
					contains(s, "state: open") &&
					contains(s, "Test body")
			},
		},
		{
			name: "body without newline",
			iss: Issue{
				FrontMatter: FrontMatter{Title: "Test"},
				Body:        "No newline",
			},
			wantErr: false,
			check: func(data []byte) bool {
				s := string(data)
				return endsWith(s, "\n")
			},
		},
		{
			name: "body with newline",
			iss: Issue{
				FrontMatter: FrontMatter{Title: "Test"},
				Body:        "Has newline\n",
			},
			wantErr: false,
			check: func(data []byte) bool {
				s := string(data)
				// Should not add extra newline
				return !endsWith(s, "\n\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Render(tt.iss)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(data) {
				t.Error("Render() result check failed")
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "issues", "123", "issue.md")

	iss := Issue{
		FrontMatter: FrontMatter{
			Number: 123,
			Title:  "Test",
		},
		Body: "Body",
	}

	if err := WriteFile(testPath, iss); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); err != nil {
		t.Errorf("file not created: %v", err)
	}

	// Verify content
	loaded, err := ParseFile(testPath)
	if err != nil {
		t.Fatalf("failed to parse written file: %v", err)
	}
	if loaded.FrontMatter.Number != 123 {
		t.Errorf("expected number 123, got %d", loaded.FrontMatter.Number)
	}
}

func TestSampleFrontMatter(t *testing.T) {
	fm := SampleFrontMatter("test issue")

	if fm.Title != "test issue" {
		t.Errorf("expected title 'test issue', got %s", fm.Title)
	}
	if len(fm.Labels) != 1 || fm.Labels[0] != "enhancement" {
		t.Errorf("expected labels [enhancement], got %v", fm.Labels)
	}
	if len(fm.Assignees) != 0 {
		t.Errorf("expected empty assignees, got %v", fm.Assignees)
	}
	if fm.Milestone != nil {
		t.Error("expected nil milestone")
	}
}

func TestSampleBody(t *testing.T) {
	body := SampleBody("test issue")

	if !contains(body, "# test issue") {
		t.Error("sample body missing title")
	}
	if !contains(body, "## 概要") {
		t.Error("sample body missing 概要 section")
	}
	if !contains(body, "## 背景") {
		t.Error("sample body missing 背景 section")
	}
	if !contains(body, "## やること") {
		t.Error("sample body missing やること section")
	}
	if !contains(body, "## 完了条件") {
		t.Error("sample body missing 完了条件 section")
	}
}

func TestTitleFromDirName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"add_login_error_handling", "Add login error handling"},
		{"fix_bug", "Fix bug"},
		{"simple", "Simple"},
		{"123_feature", "123 feature"},
		{"", ""},
		{"single", "Single"},
		{"a_b_c", "A b c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TitleFromDirName(tt.input)
			if result != tt.expected {
				t.Errorf("TitleFromDirName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSHA256Hex(t *testing.T) {
	data := []byte("test data")
	hash := SHA256Hex(data)

	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}

	// Test consistency
	hash2 := SHA256Hex(data)
	if hash != hash2 {
		t.Error("SHA256Hex not consistent")
	}

	// Test different data produces different hash
	hash3 := SHA256Hex([]byte("different data"))
	if hash == hash3 {
		t.Error("different data should produce different hash")
	}
}

func TestMeta(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, ".meta.json")

	now := time.Now()
	meta := Meta{
		Number:           123,
		ID:               "test-id",
		Title:            "Test Issue",
		State:            "open",
		URL:              "https://github.com/test/repo/issues/123",
		IsPullRequest:    false,
		UpdatedAt:        now,
		LastSyncedAt:     now,
		IssueMDSHA256:    "abc123",
		RemoteBodySHA256: "def456",
	}

	// Test SaveMeta
	if err := SaveMeta(metaPath, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	// Test LoadMeta
	loaded, err := LoadMeta(metaPath)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	if loaded.Number != meta.Number {
		t.Errorf("expected number %d, got %d", meta.Number, loaded.Number)
	}
	if loaded.ID != meta.ID {
		t.Errorf("expected id %s, got %s", meta.ID, loaded.ID)
	}
	if loaded.Title != meta.Title {
		t.Errorf("expected title %s, got %s", meta.Title, loaded.Title)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func endsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}
