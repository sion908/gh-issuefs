package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	if p.Root != "/root" {
		t.Errorf("expected Root /root, got %s", p.Root)
	}
	if p.DesignDir != "/root/.design" {
		t.Errorf("expected DesignDir /root/.design, got %s", p.DesignDir)
	}
	if p.IssuesDir != "/root/.design/issues" {
		t.Errorf("expected IssuesDir /root/.design/issues, got %s", p.IssuesDir)
	}
	if p.PrDirPath != "/root/.design/pr" {
		t.Errorf("expected PrDirPath /root/.design/pr, got %s", p.PrDirPath)
	}
	if p.ConfigPath != "/root/.design/config.toml" {
		t.Errorf("expected ConfigPath /root/.design/config.toml, got %s", p.ConfigPath)
	}
	if p.RawDir != "/root/.design/raw" {
		t.Errorf("expected RawDir /root/.design/raw, got %s", p.RawDir)
	}
}

func TestIssueDir(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.IssueDir("123_test")
	expected := "/root/.design/issues/123_test"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestIssueMDPath(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.IssueMDPath("123_test")
	expected := "/root/.design/issues/123_test/issue.md"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCommentsPath(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.CommentsPath("123_test")
	expected := "/root/.design/issues/123_test/comments.json"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMetaPath(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.MetaPath("123_test")
	expected := "/root/.design/issues/123_test/.meta.json"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRawIssuePath(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.RawIssuePath(123)
	expected := "/root/.design/raw/issues/123.json"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRawProjectPath(t *testing.T) {
	p := New("/root", ".design", "issues", "pr")

	result := p.RawProjectPath(456)
	expected := "/root/.design/raw/projects_v2/456.json"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestEnsureLayout(t *testing.T) {
	tmpDir := t.TempDir()
	p := New(tmpDir, ".design", "issues", "pr")

	if err := p.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout failed: %v", err)
	}

	// Check directories exist
	if _, err := os.Stat(p.DesignDir); err != nil {
		t.Errorf("DesignDir not created: %v", err)
	}
	if _, err := os.Stat(p.IssuesDir); err != nil {
		t.Errorf("IssuesDir not created: %v", err)
	}
	if _, err := os.Stat(p.PrDirPath); err != nil {
		t.Errorf("PrDirPath not created: %v", err)
	}
}

func TestFindDesignRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .design/config.toml structure
	designDir := filepath.Join(tmpDir, "project", ".design")
	if err := os.MkdirAll(designDir, 0o755); err != nil {
		t.Fatalf("failed to create design dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(designDir, "config.toml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Test from subdirectory
	subDir := filepath.Join(tmpDir, "project", "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	root := FindDesignRoot(subDir, ".design")
	expected := filepath.Join(tmpDir, "project")
	if root != expected {
		t.Errorf("expected %s, got %s", expected, root)
	}

	// Test when config doesn't exist
	noConfigDir := t.TempDir()
	root = FindDesignRoot(noConfigDir, ".design")
	if root != "" {
		t.Errorf("expected empty string, got %s", root)
	}
}

func TestFindGitRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .git directory
	gitDir := filepath.Join(tmpDir, "project", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Test from subdirectory
	subDir := filepath.Join(tmpDir, "project", "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	root := FindGitRoot(subDir)
	expected := filepath.Join(tmpDir, "project")
	if root != expected {
		t.Errorf("expected %s, got %s", expected, root)
	}

	// Test when .git doesn't exist
	noGitDir := t.TempDir()
	root = FindGitRoot(noGitDir)
	if root != "" {
		t.Errorf("expected empty string, got %s", root)
	}
}

func TestIssueDirNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123_add_feature", 123},
		{"456", 456},
		{"0_test", 0},
		{"9999_long_name", 9999},
		{"no_number", 0},
		{"abc_123", 0}, // non-numeric prefix
		{"", 0},
		{"1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IssueDirNumber(tt.input)
			if result != tt.expected {
				t.Errorf("IssueDirNumber(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIssueDirName(t *testing.T) {
	tests := []struct {
		number  int
		suffix  string
		expected string
	}{
		{123, "add_feature", "123_add_feature"},
		{456, "", "456"},
		{0, "test", "0_test"},
		{999, "a_b_c", "999_a_b_c"},
	}

	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			result := IssueDirName(tt.number, tt.suffix)
			if result != tt.expected {
				t.Errorf("IssueDirName(%d, %q) = %q, want %q", tt.number, tt.suffix, result, tt.expected)
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

func TestListIssueDirs(t *testing.T) {
	tmpDir := t.TempDir()
	p := New(tmpDir, ".design", "issues", "pr")

	// Create issues directory with some subdirectories
	if err := os.MkdirAll(p.IssuesDir, 0o755); err != nil {
		t.Fatalf("failed to create issues dir: %v", err)
	}

	dirs := []string{"123_test", "456_fix", "787_feature"}
	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(p.IssuesDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
	}

	// Create a file (should be ignored)
	if err := os.WriteFile(filepath.Join(p.IssuesDir, "file.txt"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result, err := p.ListIssueDirs()
	if err != nil {
		t.Fatalf("ListIssueDirs failed: %v", err)
	}

	if len(result) != len(dirs) {
		t.Errorf("expected %d dirs, got %d", len(dirs), len(result))
	}

	// Check that all expected dirs are present
	for _, expected := range dirs {
		found := false
		for _, got := range result {
			if got == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected dir %s not found in result", expected)
		}
	}
}

func TestListIssueDirs_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	p := New(tmpDir, ".design", "issues", "pr")

	// Don't create issues directory
	result, err := p.ListIssueDirs()
	if err != nil {
		t.Fatalf("ListIssueDirs failed: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for non-existent directory, got %v", result)
	}
}
