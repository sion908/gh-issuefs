package paths

import (
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	Root       string
	DesignDir  string
	IssuesDir  string
	ConfigPath string
	RawDir     string
}

func New(root, designDir, issuesDir string) Paths {
	dd := filepath.Join(root, designDir)
	id := filepath.Join(dd, issuesDir)
	return Paths{
		Root:       root,
		DesignDir:  dd,
		IssuesDir:  id,
		ConfigPath: filepath.Join(dd, "config.toml"),
		RawDir:     filepath.Join(dd, "raw"),
	}
}

func (p Paths) IssueDir(name string) string {
	return filepath.Join(p.IssuesDir, name)
}

func (p Paths) IssueMDPath(dirName string) string {
	return filepath.Join(p.IssuesDir, dirName, "issue.md")
}

func (p Paths) CommentsPath(dirName string) string {
	return filepath.Join(p.IssuesDir, dirName, "comments.json")
}

func (p Paths) MetaPath(dirName string) string {
	return filepath.Join(p.IssuesDir, dirName, ".meta.json")
}

func (p Paths) RawIssuePath(number int) string {
	return filepath.Join(p.RawDir, "issues", itoa(number)+".json")
}

func (p Paths) RawProjectPath(number int) string {
	return filepath.Join(p.RawDir, "projects_v2", itoa(number)+".json")
}

func (p Paths) EnsureLayout() error {
	for _, dir := range []string{p.DesignDir, p.IssuesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// FindDesignRoot walks upward from startDir to find a directory containing .design/config.toml.
// Returns the root directory (parent of .design), or empty string if not found.
func FindDesignRoot(startDir, designDir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, designDir, "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// FindGitRoot walks upward from startDir to find the directory containing .git.
func FindGitRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// IssueDirNumber extracts the issue number from a directory name like "123_add_hoge" or "123".
// Returns 0 if no number prefix is found.
func IssueDirNumber(name string) int {
	parts := strings.SplitN(name, "_", 2)
	n := 0
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// IssueDirName returns the directory name for a given issue number and optional suffix.
func IssueDirName(number int, suffix string) string {
	s := itoa(number)
	if suffix != "" {
		s += "_" + suffix
	}
	return s
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

// ListIssueDirs returns all subdirectory names under IssuesDir.
func (p Paths) ListIssueDirs() ([]string, error) {
	entries, err := os.ReadDir(p.IssuesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}
