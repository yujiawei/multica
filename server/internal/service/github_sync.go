package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/realtime"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// GitHubSyncService handles syncing GitHub issues with Multica issues.
type GitHubSyncService struct {
	Queries *db.Queries
	Hub     *realtime.Hub
	Bus     *events.Bus
	Client  *http.Client
}

func NewGitHubSyncService(q *db.Queries, hub *realtime.Hub, bus *events.Bus) *GitHubSyncService {
	return &GitHubSyncService{
		Queries: q,
		Hub:     hub,
		Bus:     bus,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// ghIssue is the subset of GitHub API issue fields we use.
type ghIssue struct {
	Number  int       `json:"number"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	State   string    `json:"state"`
	HTMLURL string    `json:"html_url"`
	Labels  []ghLabel `json:"labels"`
}

type ghLabel struct {
	Name string `json:"name"`
}

// SyncConfig runs a sync for a single config: fetches labeled GitHub issues
// and creates corresponding Multica issues for any not yet mapped.
func (s *GitHubSyncService) SyncConfig(ctx context.Context, config db.GithubSyncConfig) (int, error) {
	issues, err := s.fetchLabeledIssues(ctx, config)
	if err != nil {
		return 0, fmt.Errorf("fetch github issues: %w", err)
	}

	githubRepo := config.RepoOwner + "/" + config.RepoName
	created := 0

	// Fetch workspace for issue identifier prefix.
	ws, err := s.Queries.GetWorkspace(ctx, config.WorkspaceID)
	if err != nil {
		return 0, fmt.Errorf("get workspace: %w", err)
	}

	for _, ghIss := range issues {
		// Check if already mapped.
		_, err := s.Queries.GetGitHubIssueMappingByGitHub(ctx, db.GetGitHubIssueMappingByGitHubParams{
			WorkspaceID:       config.WorkspaceID,
			GithubRepo:        githubRepo,
			GithubIssueNumber: int32(ghIss.Number),
		})
		if err == nil {
			continue // already synced
		}

		multicaIssue, err := s.createMulticaIssue(ctx, config, ghIss)
		if err != nil {
			slog.Error("github sync: failed to create multica issue",
				"github_issue", ghIss.Number, "repo", githubRepo, "error", err)
			continue
		}

		_, err = s.Queries.CreateGitHubIssueMapping(ctx, db.CreateGitHubIssueMappingParams{
			WorkspaceID:       config.WorkspaceID,
			ConfigID:          config.ID,
			GithubRepo:        githubRepo,
			GithubIssueNumber: int32(ghIss.Number),
			GithubIssueUrl:    ghIss.HTMLURL,
			MulticaIssueID:    multicaIssue.ID,
		})
		if err != nil {
			slog.Error("github sync: failed to create mapping",
				"github_issue", ghIss.Number, "multica_issue", util.UUIDToString(multicaIssue.ID), "error", err)
			continue
		}

		// Post a comment on the GitHub issue linking back to the Multica issue.
		identifier := fmt.Sprintf("%s-%d", ws.IssuePrefix, multicaIssue.Number)
		if err := s.postGitHubComment(ctx, config, ghIss.Number, identifier); err != nil {
			slog.Error("github sync: failed to post github comment",
				"github_issue", ghIss.Number, "error", err)
			// Non-fatal: the issue was created successfully, just the comment failed.
		}

		created++
		slog.Info("github sync: created multica issue",
			"github_issue", ghIss.Number, "multica_issue", util.UUIDToString(multicaIssue.ID), "repo", githubRepo)
	}

	s.Queries.UpdateGitHubSyncConfigLastSynced(ctx, config.ID)
	return created, nil
}

// SyncAll runs a sync for all active configs.
func (s *GitHubSyncService) SyncAll(ctx context.Context) error {
	configs, err := s.Queries.ListActiveGitHubSyncConfigs(ctx)
	if err != nil {
		return fmt.Errorf("list active configs: %w", err)
	}

	for _, config := range configs {
		if _, err := s.SyncConfig(ctx, config); err != nil {
			slog.Error("github sync: config sync failed",
				"config_id", util.UUIDToString(config.ID),
				"repo", config.RepoOwner+"/"+config.RepoName,
				"error", err)
		}
	}
	return nil
}

// CloseGitHubIssue closes the corresponding GitHub issue when a Multica issue
// reaches "done" status. This is the reverse direction of the sync.
func (s *GitHubSyncService) CloseGitHubIssue(ctx context.Context, multicaIssueID pgtype.UUID) error {
	mapping, err := s.Queries.GetGitHubIssueMappingByMulticaIssue(ctx, multicaIssueID)
	if err != nil {
		return nil // no mapping, nothing to do
	}

	config, err := s.Queries.GetGitHubSyncConfig(ctx, mapping.ConfigID)
	if err != nil {
		return fmt.Errorf("load sync config: %w", err)
	}
	if !config.GithubToken.Valid || config.GithubToken.String == "" {
		slog.Debug("github sync: no token configured, skipping close", "config_id", util.UUIDToString(config.ID))
		return nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d",
		config.RepoOwner, config.RepoName, mapping.GithubIssueNumber)

	body := `{"state":"closed"}`
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.GithubToken.String)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("close github issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API error %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("github sync: closed github issue",
		"repo", mapping.GithubRepo, "issue", mapping.GithubIssueNumber)
	return nil
}

func (s *GitHubSyncService) fetchLabeledIssues(ctx context.Context, config db.GithubSyncConfig) ([]ghIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?labels=%s&state=open&per_page=100",
		config.RepoOwner, config.RepoName, config.LabelFilter)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if config.GithubToken.Valid && config.GithubToken.String != "" {
		req.Header.Set("Authorization", "Bearer "+config.GithubToken.String)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var issues []ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return issues, nil
}

func (s *GitHubSyncService) createMulticaIssue(ctx context.Context, config db.GithubSyncConfig, ghIss ghIssue) (db.Issue, error) {
	// Increment issue counter.
	counter, err := s.Queries.IncrementIssueCounter(ctx, config.WorkspaceID)
	if err != nil {
		return db.Issue{}, fmt.Errorf("increment counter: %w", err)
	}

	// Build description with link back to GitHub issue.
	desc := ghIss.Body
	if ghIss.HTMLURL != "" {
		desc = fmt.Sprintf("GitHub: %s\n\n%s", ghIss.HTMLURL, desc)
	}

	title := ghIss.Title
	// Include GitHub issue number in title for traceability.
	title = fmt.Sprintf("%s (#%d)", title, ghIss.Number)

	params := db.CreateIssueParams{
		WorkspaceID: config.WorkspaceID,
		Title:       title,
		Description: pgtype.Text{String: desc, Valid: desc != ""},
		Status:      "todo",
		Priority:    "medium",
		CreatorType: "agent",
		Number:      counter,
	}

	// Use a system creator if no default agent — we still need a valid creator.
	// If default_agent_id is set, assign and set as creator.
	if config.DefaultAgentID.Valid {
		params.AssigneeType = pgtype.Text{String: "agent", Valid: true}
		params.AssigneeID = config.DefaultAgentID
		params.CreatorID = config.DefaultAgentID
	} else {
		// Use workspace creator as fallback. Look up the first owner member.
		params.CreatorType = "member"
		params.CreatorID = findWorkspaceOwner(ctx, s.Queries, config.WorkspaceID)
	}

	return s.Queries.CreateIssue(ctx, params)
}

// postGitHubComment posts a comment on a GitHub issue linking to the Multica issue.
func (s *GitHubSyncService) postGitHubComment(ctx context.Context, config db.GithubSyncConfig, ghIssueNumber int, multicaIdentifier string) error {
	if !config.GithubToken.Valid || config.GithubToken.String == "" {
		slog.Debug("github sync: no token configured, skipping comment", "config_id", util.UUIDToString(config.ID))
		return nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments",
		config.RepoOwner, config.RepoName, ghIssueNumber)

	commentBody := fmt.Sprintf(`{"body":"🤖 Multica issue created: **%s**\n\nThis issue is being tracked and will be worked on by an AI agent."}`, multicaIdentifier)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(commentBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.GithubToken.String)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("post github comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API error %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("github sync: posted comment on github issue",
		"repo", config.RepoOwner+"/"+config.RepoName, "issue", ghIssueNumber, "multica_id", multicaIdentifier)
	return nil
}

// findWorkspaceOwner returns the ID of the first owner member, or a zero UUID.
func findWorkspaceOwner(ctx context.Context, q *db.Queries, workspaceID pgtype.UUID) pgtype.UUID {
	members, err := q.ListMembers(ctx, workspaceID)
	if err != nil {
		return pgtype.UUID{}
	}
	for _, m := range members {
		if m.Role == "owner" {
			return m.UserID
		}
	}
	if len(members) > 0 {
		return members[0].UserID
	}
	return pgtype.UUID{}
}

// RunPoller starts a background goroutine that periodically syncs all active configs.
func (s *GitHubSyncService) RunPoller(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("github sync poller started", "interval", interval)

	// Run once immediately on start.
	if err := s.SyncAll(ctx); err != nil {
		slog.Error("github sync: initial sync failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("github sync poller stopped")
			return
		case <-ticker.C:
			if err := s.SyncAll(ctx); err != nil {
				slog.Error("github sync: periodic sync failed", "error", err)
			}
		}
	}
}

