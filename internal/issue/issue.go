package issue

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ProjectV2Item struct {
	Name      string `yaml:"name"`
	Status    string `yaml:"status,omitempty"`
	Iteration string `yaml:"iteration,omitempty"`
}

type FrontMatter struct {
	Number     int             `yaml:"number,omitempty"`
	Title      string          `yaml:"title"`
	State      string          `yaml:"state,omitempty"`
	Labels     []string        `yaml:"labels,omitempty"`
	Assignees  []string        `yaml:"assignees,omitempty"`
	Milestone  *string         `yaml:"milestone"`
	ProjectsV2 []ProjectV2Item `yaml:"projects_v2,omitempty"`
}

type Issue struct {
	FrontMatter FrontMatter
	Body        string
}

var separator = "---"

func Parse(data []byte) (Issue, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return Issue{Body: content}, nil
	}

	rest := content[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return Issue{}, fmt.Errorf("unclosed frontmatter")
	}

	fmRaw := rest[:end]
	body := strings.TrimPrefix(rest[end+5:], "\n")

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return Issue{}, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return Issue{FrontMatter: fm, Body: body}, nil
}

func ParseFile(path string) (Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Issue{}, err
	}
	return Parse(data)
}

func Render(iss Issue) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("---\n")
	fm, err := yaml.Marshal(iss.FrontMatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	buf.Write(fm)
	buf.WriteString("---\n")
	if iss.Body != "" {
		buf.WriteString(iss.Body)
		if !strings.HasSuffix(iss.Body, "\n") {
			buf.WriteByte('\n')
		}
	}

	return buf.Bytes(), nil
}

func WriteFile(path string, iss Issue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := Render(iss)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SampleFrontMatter returns a FrontMatter suitable for a new issue template.
func SampleFrontMatter(title string) FrontMatter {
	emptyLabels := []string{}
	emptyAssignees := []string{}
	nilMilestone := (*string)(nil)
	return FrontMatter{
		Title:     title,
		Labels:    emptyLabels,
		Assignees: emptyAssignees,
		Milestone: nilMilestone,
	}
}

// SampleBody returns a template body for a new issue.
func SampleBody(title string) string {
	return fmt.Sprintf("# %s\n## 概要\n## 背景\n## やること\n- [ ]\n## 完了条件\n- [ ]\n", title)
}

// TitleFromDirName converts a directory name like "add_login_error_handling" to
// a human-readable title like "Add login error handling".
func TitleFromDirName(name string) string {
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return name
	}
	if len(parts[0]) > 0 {
		parts[0] = strings.ToUpper(parts[0][:1]) + parts[0][1:]
	}
	return strings.Join(parts, " ")
}

// RemoteIssue is a GitHub Issue fetched from the API.
type RemoteIssue struct {
	Number        int
	ID            string
	Title         string
	Body          string
	State         string
	URL           string
	IsPullRequest bool
	UpdatedAt     time.Time
	Labels        []string
	Assignees     []string
	Milestone     *string
	ProjectsV2    []ProjectV2Item
}

// Comment is a single GitHub Issue comment.
type Comment struct {
	ID        string
	Author    string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	URL       string
}

// separator は yaml.Marshal が生成する区切り文字列を使う用途で参照する
var _ = separator
