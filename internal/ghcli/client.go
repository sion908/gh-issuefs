package ghcli

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sion908/gh-issuefs/internal/issue"
)

type Client struct {
	runner Runner
	repo   string // "owner/repo"
}

func NewClient(runner Runner, repo string) *Client {
	return &Client{runner: runner, repo: repo}
}

// DetectRepo detects the current repository from git remote via gh.
func DetectRepo(ctx context.Context, runner Runner) (string, error) {
	out, err := runner.Run(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to detect repository: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// --- REST types ---

type apiLabel struct {
	Name string `json:"name"`
}

type apiUser struct {
	Login string `json:"login"`
}

type apiMilestone struct {
	Title string `json:"title"`
}

type apiComment struct {
	NodeID    string `json:"node_id"`
	User      apiUser `json:"user"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	HTMLURL   string `json:"html_url"`
}

// --- Issue fetch ---

// GetIssue fetches a single issue by number.
func (c *Client) GetIssue(ctx context.Context, number int) (issue.RemoteIssue, error) {
	owner, repo := splitRepo(c.repo)
	query := `query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    issue(number: $number) {
      number
      title
      body
      state
      url
      isPullRequest: __typename
      updatedAt
      labels(first: 100) { nodes { name } }
      assignees(first: 100) { nodes { login } }
      milestone { title }
      projectItems(first: 20) {
        nodes {
          project { title }
          status: fieldValueByName(name: "Status") {
            ... on ProjectV2ItemFieldSingleSelectValue { name }
          }
          iteration: fieldValueByName(name: "Iteration") {
            ... on ProjectV2ItemFieldIterationValue { title }
          }
        }
      }
    }
  }
}`
	args := []string{"api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
		"-F", fmt.Sprintf("owner=%s", owner),
		"-F", fmt.Sprintf("repo=%s", repo),
		"-F", fmt.Sprintf("number=%d", number),
	}
	out, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return issue.RemoteIssue{}, err
	}
	return parseIssueGraphQL(out)
}

// ListIssues fetches issues matching the given search query using REST API.
// query example: "assignee:@me state:open"
func (c *Client) ListIssues(ctx context.Context, searchQuery string) ([]issue.RemoteIssue, error) {
	args := []string{"issue", "list",
		"--search", searchQuery,
		"--limit", "1000",
		"--json", "number,title,body,state,url,updatedAt,labels,assignees,milestone",
		"--repo", c.repo,
	}
	out, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, err
	}

	var apiIssues []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		State     string `json:"state"`
		URL       string `json:"url"`
		UpdatedAt string `json:"updatedAt"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		Milestone *struct {
			Title string `json:"title"`
		} `json:"milestone"`
	}
	if err := json.Unmarshal([]byte(out), &apiIssues); err != nil {
		return nil, err
	}

	var results []issue.RemoteIssue
	for _, ai := range apiIssues {
		labels := make([]string, 0, len(ai.Labels))
		for _, l := range ai.Labels {
			labels = append(labels, l.Name)
		}
		assignees := make([]string, 0, len(ai.Assignees))
		for _, a := range ai.Assignees {
			assignees = append(assignees, a.Login)
		}
		var milestone *string
		if ai.Milestone != nil {
			m := ai.Milestone.Title
			milestone = &m
		}
		updatedAt, _ := time.Parse(time.RFC3339, ai.UpdatedAt)

		results = append(results, issue.RemoteIssue{
			Number:    ai.Number,
			Title:     ai.Title,
			Body:      ai.Body,
			State:     strings.ToLower(ai.State),
			URL:       ai.URL,
			UpdatedAt: updatedAt,
			Labels:    labels,
			Assignees: assignees,
			Milestone: milestone,
		})
	}
	return results, nil
}

// GetComments fetches comments for an issue.
func (c *Client) GetComments(ctx context.Context, number int) ([]issue.Comment, error) {
	owner, repo := splitRepo(c.repo)
	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number)
	args := []string{"api", endpoint, "--paginate"}
	out, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, err
	}

	// --paginate produces concatenated JSON arrays; wrap them
	merged, err := mergeJSONArrays(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	var raw []apiComment
	if err := json.Unmarshal(merged, &raw); err != nil {
		return nil, err
	}

	comments := make([]issue.Comment, 0, len(raw))
	for _, c := range raw {
		var createdAt, updatedAt time.Time
		createdAt, _ = time.Parse(time.RFC3339, c.CreatedAt)
		updatedAt, _ = time.Parse(time.RFC3339, c.UpdatedAt)
		comments = append(comments, issue.Comment{
			ID:        c.NodeID,
			Author:    c.User.Login,
			Body:      c.Body,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			URL:       c.HTMLURL,
		})
	}
	return comments, nil
}

// CreateIssue creates a new GitHub issue and returns its number and node ID.
func (c *Client) CreateIssue(ctx context.Context, title, body string) (int, string, error) {
	owner, repo := splitRepo(c.repo)
	args := []string{"issue", "create",
		"--repo", owner + "/" + repo,
		"--title", title,
		"--body", body,
	}
	out, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return 0, "", err
	}
	num, err := parseIssueNumberFromURL(strings.TrimSpace(out))
	if err != nil {
		return 0, "", err
	}
	// Fetch the node ID
	ri, err := c.GetIssue(ctx, num)
	if err != nil {
		return num, "", nil
	}
	return num, ri.ID, nil
}

// EditIssue updates title and/or body of an existing issue.
func (c *Client) EditIssue(ctx context.Context, number int, title, body string) error {
	owner, repo := splitRepo(c.repo)
	args := []string{"issue", "edit", strconv.Itoa(number),
		"--repo", owner + "/" + repo,
		"--title", title,
		"--body", body,
	}
	_, err := c.runner.Run(ctx, "gh", args...)
	return err
}

// --- internal helpers ---

type projectItemNode struct {
	Project struct {
		Title string `json:"title"`
	} `json:"project"`
	Status *struct {
		Name string `json:"name"`
	} `json:"status"`
	Iteration *struct {
		Title string `json:"title"`
	} `json:"iteration"`
}

type issueNode struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	State    string `json:"state"`
	URL      string `json:"url"`
	Typename string `json:"__typename"`
	UpdatedAt string `json:"updatedAt"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	ProjectItems *struct {
		Nodes []projectItemNode `json:"nodes"`
	} `json:"projectItems"`
}

func issueNodeToRemote(node issueNode) (issue.RemoteIssue, error) {
	labels := make([]string, 0, len(node.Labels.Nodes))
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}
	assignees := make([]string, 0, len(node.Assignees.Nodes))
	for _, a := range node.Assignees.Nodes {
		assignees = append(assignees, a.Login)
	}
	var milestone *string
	if node.Milestone != nil {
		m := node.Milestone.Title
		milestone = &m
	}

	var projects []issue.ProjectV2Item
	if node.ProjectItems != nil {
		for _, pi := range node.ProjectItems.Nodes {
			item := issue.ProjectV2Item{Name: pi.Project.Title}
			if pi.Status != nil {
				item.Status = pi.Status.Name
			}
			if pi.Iteration != nil {
				item.Iteration = pi.Iteration.Title
			}
			projects = append(projects, item)
		}
	}

	updatedAt, _ := time.Parse(time.RFC3339, node.UpdatedAt)

	return issue.RemoteIssue{
		Number:        node.Number,
		Title:         node.Title,
		Body:          node.Body,
		State:         strings.ToLower(node.State),
		URL:           node.URL,
		IsPullRequest: node.Typename == "PullRequest",
		UpdatedAt:     updatedAt,
		Labels:        labels,
		Assignees:     assignees,
		Milestone:     milestone,
		ProjectsV2:    projects,
	}, nil
}

func parseIssueGraphQL(out string) (issue.RemoteIssue, error) {
	var resp struct {
		Data struct {
			Repository struct {
				Issue *issueNode `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return issue.RemoteIssue{}, err
	}
	if len(resp.Errors) > 0 {
		return issue.RemoteIssue{}, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
	}
	if resp.Data.Repository.Issue == nil {
		return issue.RemoteIssue{}, fmt.Errorf("issue not found")
	}
	return issueNodeToRemote(*resp.Data.Repository.Issue)
}

var issueURLPattern = regexp.MustCompile(`/issues/(\d+)`)

func parseIssueNumberFromURL(s string) (int, error) {
	m := issueURLPattern.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, fmt.Errorf("unable to parse issue number from: %q", s)
	}
	return strconv.Atoi(m[1])
}

func splitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// parseSearchQuery extracts state and assignee from a simple search query string.
func parseSearchQuery(q string) (states []string, assignee string) {
	for _, token := range strings.Fields(q) {
		kv := strings.SplitN(token, ":", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "state":
			states = append(states, kv[1])
		case "assignee":
			assignee = kv[1]
		}
	}
	if len(states) == 0 {
		states = []string{"open"}
	}
	return
}

// mergeJSONArrays merges multiple concatenated JSON arrays (from --paginate) into one.
func mergeJSONArrays(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []byte("[]"), nil
	}
	// gh --paginate outputs multiple arrays concatenated; combine them
	s = strings.ReplaceAll(s, "][", ",")
	return []byte(s), nil
}
