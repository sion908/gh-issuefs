package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Data            DataConfig      `toml:"data"`
	Sync            SyncConfig      `toml:"sync"`
	Storage         StorageConfig   `toml:"storage"`
	Formatter       FormatterConfig `toml:"formatter"`
	Push            PushConfig      `toml:"push"`
	DefaultTemplate string          `toml:"default_template"`
}

type DataConfig struct {
	RootDir   string `toml:"root_dir"`
	IssuesDir string `toml:"issues_dir"`
}

type SyncConfig struct {
	DefaultQuery         string `toml:"default_query"`
	IncludeComments      bool   `toml:"include_comments"`
	ExcludePullRequests  bool   `toml:"exclude_pull_requests"`
}

type StorageConfig struct {
	SaveRaw bool `toml:"save_raw"`
}

type FormatterConfig struct {
	IncludeLabels      bool `toml:"include_labels"`
	IncludeMilestone   bool `toml:"include_milestone"`
	IncludeAssignees   bool `toml:"include_assignees"`
	IncludeProjectsV2  bool `toml:"include_projects_v2"`
}

type PushConfig struct {
	IssueTitle bool `toml:"issue_title"`
	IssueBody  bool `toml:"issue_body"`
	Comments   bool `toml:"comments"`
	Labels     bool `toml:"labels"`
	Milestone  bool `toml:"milestone"`
	Assignees  bool `toml:"assignees"`
	ProjectsV2 bool `toml:"projects_v2"`
}

func Default() Config {
	return Config{
		Data: DataConfig{
			RootDir:   ".design",
			IssuesDir: "issues",
		},
		Sync: SyncConfig{
			DefaultQuery:        "assignee:@me state:open",
			IncludeComments:     true,
			ExcludePullRequests: true,
		},
		Storage: StorageConfig{
			SaveRaw: false,
		},
		Formatter: FormatterConfig{
			IncludeLabels:     true,
			IncludeMilestone:  true,
			IncludeAssignees:  true,
			IncludeProjectsV2: true,
		},
		Push: PushConfig{
			IssueTitle: true,
			IssueBody:  true,
			Comments:   false,
			Labels:     false,
			Milestone:  false,
			Assignees:  false,
			ProjectsV2: false,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config: %w", err)
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func GlobalConfigPath() string {
	dir := os.Getenv("GH_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config", "gh")
	}
	return filepath.Join(dir, "gh-design", "config.toml")
}

func LoadWithFallback(localPath string) (Config, error) {
	if _, err := os.Stat(localPath); err == nil {
		return Load(localPath)
	}
	globalPath := GlobalConfigPath()
	if _, err := os.Stat(globalPath); err == nil {
		return Load(globalPath)
	}
	return Default(), nil
}
