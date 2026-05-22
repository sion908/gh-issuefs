package ghcli

import (
	"context"
	"errors"
	"testing"
)

// MockRunner is a mock implementation of Runner for testing
type MockRunner struct {
	Output string
	Err    error
}

func (m *MockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return m.Output, m.Err
}


func TestNewClient(t *testing.T) {
	// NewClient の動作は各 API テスト（GetIssue 等）で間接的に検証される。
	// ここではゼロ値 runner での初期化が panic しないことだけ確認する。
	runner := &MockRunner{}
	_ = NewClient(runner, "owner/repo")
}

func TestDetectRepo(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		runner := &MockRunner{Output: "owner/repo\n"}
		repo, err := DetectRepo(ctx, runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo != "owner/repo" {
			t.Errorf("expected owner/repo, got %s", repo)
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		_, err := DetectRepo(ctx, runner)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		graphQLResp := `{
			"data": {
				"repository": {
					"issueOrPullRequest": {
						"number": 123,
						"title": "Test Issue",
						"body": "Test body",
						"state": "OPEN",
						"url": "https://github.com/owner/repo/issues/123",
						"typename": "Issue",
						"updatedAt": "2024-01-01T00:00:00Z",
						"labels": {
							"nodes": [{"name": "bug"}]
						},
						"assignees": {
							"nodes": [{"login": "user1"}]
						},
						"milestone": {"title": "v1.0"},
						"projectItems": {
							"nodes": []
						}
					}
				}
			}
		}`
		runner := &MockRunner{Output: graphQLResp}
		client := NewClient(runner, "owner/repo")

		ri, err := client.GetIssue(ctx, 123)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ri.Number != 123 {
			t.Errorf("expected number 123, got %d", ri.Number)
		}
		if ri.Title != "Test Issue" {
			t.Errorf("expected title Test Issue, got %s", ri.Title)
		}
		if ri.State != "open" {
			t.Errorf("expected state open, got %s", ri.State)
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		client := NewClient(runner, "owner/repo")

		_, err := client.GetIssue(ctx, 123)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestListIssues(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		issuesResp := `[{
			"number": 123,
			"title": "Test Issue",
			"body": "Test body",
			"state": "OPEN",
			"url": "https://github.com/owner/repo/issues/123",
			"updatedAt": "2024-01-01T00:00:00Z",
			"labels": [{"name": "bug"}],
			"assignees": [{"login": "user1"}],
			"milestone": {"title": "v1.0"}
		}]`
		runner := &MockRunner{Output: issuesResp}
		client := NewClient(runner, "owner/repo")

		issues, err := client.ListIssues(ctx, "assignee:@me")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		if issues[0].Number != 123 {
			t.Errorf("expected number 123, got %d", issues[0].Number)
		}
		if issues[0].State != "open" {
			t.Errorf("expected state open, got %s", issues[0].State)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		runner := &MockRunner{Output: "[]"}
		client := NewClient(runner, "owner/repo")

		issues, err := client.ListIssues(ctx, "assignee:@me")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		client := NewClient(runner, "owner/repo")

		_, err := client.ListIssues(ctx, "assignee:@me")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetComments(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		commentsResp := `[{
			"node_id": "node123",
			"user": {"login": "user1"},
			"body": "Test comment",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"html_url": "https://github.com/owner/repo/issues/123#comment-123"
		}]`
		runner := &MockRunner{Output: commentsResp}
		client := NewClient(runner, "owner/repo")

		comments, err := client.GetComments(ctx, 123, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].ID != "node123" {
			t.Errorf("expected id node123, got %s", comments[0].ID)
		}
		if comments[0].Author != "user1" {
			t.Errorf("expected author user1, got %s", comments[0].Author)
		}
	})

	t.Run("pull request comments", func(t *testing.T) {
		commentsResp := `[{
			"node_id": "node456",
			"user": {"login": "user2"},
			"body": "PR comment",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"html_url": "https://github.com/owner/repo/pull/456#comment-456"
		}]`
		runner := &MockRunner{Output: commentsResp}
		client := NewClient(runner, "owner/repo")

		comments, err := client.GetComments(ctx, 456, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].ID != "node456" {
			t.Errorf("expected id node456, got %s", comments[0].ID)
		}
	})

	t.Run("paginated response", func(t *testing.T) {
		// Simulate --paginate output (concatenated arrays)
		commentsResp := `[{"node_id":"n1","user":{"login":"u1"},"body":"c1","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","html_url":"url1"}][{"node_id":"n2","user":{"login":"u2"},"body":"c2","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","html_url":"url2"}]`
		runner := &MockRunner{Output: commentsResp}
		client := NewClient(runner, "owner/repo")

		comments, err := client.GetComments(ctx, 123, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(comments) != 2 {
			t.Fatalf("expected 2 comments, got %d", len(comments))
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		client := NewClient(runner, "owner/repo")

		_, err := client.GetComments(ctx, 123, false)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestCreateIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		// Mock create issue response
		createResp := "https://github.com/owner/repo/issues/456\n"
		runner := &MockRunner{Output: createResp}
		client := NewClient(runner, "owner/repo")
		
		num, _, err := client.CreateIssue(ctx, "New Issue", "Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if num != 456 {
			t.Errorf("expected number 456, got %d", num)
		}
	})

	t.Run("create error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		client := NewClient(runner, "owner/repo")

		_, _, err := client.CreateIssue(ctx, "New Issue", "Body")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestEditIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		runner := &MockRunner{Output: ""}
		client := NewClient(runner, "owner/repo")

		err := client.EditIssue(ctx, 123, "Updated Title", "Updated Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		runner := &MockRunner{Err: errors.New("gh failed")}
		client := NewClient(runner, "owner/repo")

		err := client.EditIssue(ctx, 123, "Title", "Body")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestParseIssueOrPRGraphQL(t *testing.T) {
	t.Run("valid issue response", func(t *testing.T) {
		resp := `{
			"data": {
				"repository": {
					"issueOrPullRequest": {
						"number": 123,
						"title": "Test",
						"body": "Body",
						"state": "OPEN",
						"url": "https://github.com/owner/repo/issues/123",
						"typename": "Issue",
						"updatedAt": "2024-01-01T00:00:00Z",
						"labels": {"nodes": []},
						"assignees": {"nodes": []},
						"milestone": null,
						"projectItems": {"nodes": []}
					}
				}
			}
		}`
		ri, err := parseIssueOrPRGraphQL(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ri.Number != 123 {
			t.Errorf("expected number 123, got %d", ri.Number)
		}
		if ri.IsPullRequest {
			t.Error("expected IsPullRequest false for Issue")
		}
	})

	t.Run("valid pull request response", func(t *testing.T) {
		resp := `{
			"data": {
				"repository": {
					"issueOrPullRequest": {
						"number": 169,
						"title": "Test PR",
						"body": "PR Body",
						"state": "OPEN",
						"url": "https://github.com/owner/repo/pull/169",
						"typename": "PullRequest",
						"updatedAt": "2024-01-01T00:00:00Z",
						"labels": {"nodes": []},
						"assignees": {"nodes": []},
						"milestone": null,
						"projectItems": {"nodes": []}
					}
				}
			}
		}`
		ri, err := parseIssueOrPRGraphQL(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ri.Number != 169 {
			t.Errorf("expected number 169, got %d", ri.Number)
		}
		if !ri.IsPullRequest {
			t.Error("expected IsPullRequest true for PullRequest")
		}
	})

	t.Run("GraphQL error", func(t *testing.T) {
		resp := `{
			"data": {"repository": {"issueOrPullRequest": null}},
			"errors": [{"message": "Not found"}]
		}`
		_, err := parseIssueOrPRGraphQL(resp)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp := `{
			"data": {"repository": {"issueOrPullRequest": null}}
		}`
		_, err := parseIssueOrPRGraphQL(resp)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestParseIssueNumberFromURL(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"https://github.com/owner/repo/issues/123", 123, false},
		{"https://github.com/owner/repo/issues/456#comment-789", 456, false},
		{"https://github.com/owner/repo/issues/999", 999, false},
		{"invalid url", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			num, err := parseIssueNumberFromURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if num != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, num)
			}
		})
	}
}

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		input       string
		expectedOwner string
		expectedRepo  string
	}{
		{"owner/repo", "owner", "repo"},
		{"org/project", "org", "project"},
		{"invalid", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo := splitRepo(tt.input)
			if owner != tt.expectedOwner {
				t.Errorf("expected owner %s, got %s", tt.expectedOwner, owner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("expected repo %s, got %s", tt.expectedRepo, repo)
			}
		})
	}
}

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		input          string
		expectedStates []string
		expectedAssignee string
	}{
		{"assignee:@me state:open", []string{"open"}, "@me"},
		{"state:closed state:open", []string{"closed", "open"}, ""},
		{"assignee:user1", []string{"open"}, "user1"},
		{"", []string{"open"}, ""},
		{"random tokens", []string{"open"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			states, assignee := parseSearchQuery(tt.input)
			if len(states) != len(tt.expectedStates) {
				t.Errorf("expected %d states, got %d", len(tt.expectedStates), len(states))
			}
			for i, s := range states {
				if s != tt.expectedStates[i] {
					t.Errorf("state %d: expected %s, got %s", i, tt.expectedStates[i], s)
				}
			}
			if assignee != tt.expectedAssignee {
				t.Errorf("expected assignee %s, got %s", tt.expectedAssignee, assignee)
			}
		})
	}
}

func TestMergeJSONArrays(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"[1][2]", "[1,2]", false},
		{"[1,2][3,4]", "[1,2,3,4]", false},
		{"[]", "[]", false},
		{"", "[]", false},
		{"  [1]  ", "[1]", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := mergeJSONArrays(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

func TestIssueNodeToRemote(t *testing.T) {
	// Test basic conversion with empty collections
	node := issueNode{
		Number:    123,
		Title:     "Test",
		Body:      "Body",
		State:     "OPEN",
		URL:       "https://github.com/owner/repo/issues/123",
		Typename:  "Issue",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Labels:    struct{ Nodes []struct{ Name string `json:"name"` } `json:"nodes"` }{Nodes: []struct{ Name string `json:"name"` }{}},
		Assignees: struct{ Nodes []struct{ Login string `json:"login"` } `json:"nodes"` }{Nodes: []struct{ Login string `json:"login"` }{}},
		Milestone: nil,
		ProjectItems: &struct {
			Nodes []projectItemNode `json:"nodes"`
		}{Nodes: []projectItemNode{}},
	}

	ri, err := issueNodeToRemote(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ri.Number != 123 {
		t.Errorf("expected number 123, got %d", ri.Number)
	}
	if ri.State != "open" {
		t.Errorf("expected state open, got %s", ri.State)
	}
	if len(ri.Labels) != 0 {
		t.Errorf("expected 0 labels, got %d", len(ri.Labels))
	}
}
