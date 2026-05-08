package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Data.RootDir != ".design" {
		t.Errorf("expected RootDir .design, got %s", cfg.Data.RootDir)
	}
	if cfg.Data.IssuesDir != "issues" {
		t.Errorf("expected IssuesDir issues, got %s", cfg.Data.IssuesDir)
	}
	if cfg.Sync.DefaultQuery != "assignee:@me state:open" {
		t.Errorf("expected DefaultQuery assignee:@me state:open, got %s", cfg.Sync.DefaultQuery)
	}
	if !cfg.Sync.IncludeComments {
		t.Error("expected IncludeComments true")
	}
	if cfg.Sync.ExcludePullRequests {
		t.Error("expected ExcludePullRequests false")
	}
	if cfg.Storage.SaveRaw {
		t.Error("expected SaveRaw false")
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Save default config
	cfg := Default()
	cfg.DefaultTemplate = "default.md"
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.DefaultTemplate != "default.md" {
		t.Errorf("expected DefaultTemplate default.md, got %s", loaded.DefaultTemplate)
	}
	if loaded.Data.RootDir != cfg.Data.RootDir {
		t.Errorf("expected RootDir %s, got %s", cfg.Data.RootDir, loaded.Data.RootDir)
	}
}

func TestLoadInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	// Write invalid TOML
	if err := os.WriteFile(configPath, []byte("invalid [toml"), 0o644); err != nil {
		t.Fatalf("failed to write invalid toml: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.toml")

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	// Test with GH_CONFIG_DIR set
	oldHome := os.Getenv("HOME")
	oldConfigDir := os.Getenv("GH_CONFIG_DIR")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("GH_CONFIG_DIR", oldConfigDir)
	}()

	os.Setenv("GH_CONFIG_DIR", "/test/config")
	path := GlobalConfigPath()
	expected := "/test/config/gh-design/config.toml"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}

	// Test without GH_CONFIG_DIR
	os.Unsetenv("GH_CONFIG_DIR")
	os.Setenv("HOME", "/home/user")
	path = GlobalConfigPath()
	expected = "/home/user/.config/gh/gh-design/config.toml"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestLoadWithFallback(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, ".design", "config.toml")

	// Test fallback to default when no config exists
	cfg, err := LoadWithFallback(localPath)
	if err != nil {
		t.Fatalf("LoadWithFallback failed: %v", err)
	}
	if cfg.Data.RootDir != ".design" {
		t.Errorf("expected default RootDir, got %s", cfg.Data.RootDir)
	}

	// Test loading local config
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	cfg.DefaultTemplate = "local.md"
	if err := Save(localPath, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadWithFallback(localPath)
	if err != nil {
		t.Fatalf("LoadWithFallback failed: %v", err)
	}
	if loaded.DefaultTemplate != "local.md" {
		t.Errorf("expected local template, got %s", loaded.DefaultTemplate)
	}
}
