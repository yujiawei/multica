package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// GitHubSyncConfigResponse is the JSON response for a sync config.
type GitHubSyncConfigResponse struct {
	ID             string  `json:"id"`
	WorkspaceID    string  `json:"workspace_id"`
	RepoOwner      string  `json:"repo_owner"`
	RepoName       string  `json:"repo_name"`
	LabelFilter    string  `json:"label_filter"`
	DefaultAgentID *string `json:"default_agent_id"`
	HasToken       bool    `json:"has_token"`
	Active         bool    `json:"active"`
	LastSyncedAt   *string `json:"last_synced_at"`
	CreatedAt      string  `json:"created_at"`
}

func syncConfigToResponse(c db.GithubSyncConfig) GitHubSyncConfigResponse {
	return GitHubSyncConfigResponse{
		ID:             uuidToString(c.ID),
		WorkspaceID:    uuidToString(c.WorkspaceID),
		RepoOwner:      c.RepoOwner,
		RepoName:       c.RepoName,
		LabelFilter:    c.LabelFilter,
		DefaultAgentID: uuidToPtr(c.DefaultAgentID),
		HasToken:       c.GithubToken.Valid && c.GithubToken.String != "",
		Active:         c.Active,
		LastSyncedAt:   timestampToPtr(c.LastSyncedAt),
		CreatedAt:      timestampToString(c.CreatedAt),
	}
}

// ListGitHubSyncConfigs lists all sync configs for the workspace.
func (h *Handler) ListGitHubSyncConfigs(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}

	configs, err := h.Queries.ListGitHubSyncConfigs(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list configs")
		return
	}

	resp := make([]GitHubSyncConfigResponse, len(configs))
	for i, c := range configs {
		resp[i] = syncConfigToResponse(c)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateGitHubSyncConfig creates a new sync config.
func (h *Handler) CreateGitHubSyncConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}

	var req struct {
		RepoOwner      string  `json:"repo_owner"`
		RepoName       string  `json:"repo_name"`
		LabelFilter    string  `json:"label_filter"`
		DefaultAgentID *string `json:"default_agent_id"`
		GitHubToken    *string `json:"github_token"`
		Active         *bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RepoOwner == "" || req.RepoName == "" {
		writeError(w, http.StatusBadRequest, "repo_owner and repo_name are required")
		return
	}

	labelFilter := "multica"
	if req.LabelFilter != "" {
		labelFilter = req.LabelFilter
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	config, err := h.Queries.CreateGitHubSyncConfig(r.Context(), db.CreateGitHubSyncConfigParams{
		WorkspaceID:    parseUUID(workspaceID),
		RepoOwner:      req.RepoOwner,
		RepoName:       req.RepoName,
		LabelFilter:    labelFilter,
		DefaultAgentID: parseOptionalUUID(req.DefaultAgentID),
		GithubToken:    ptrToText(req.GitHubToken),
		Active:         active,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "sync config already exists for this repo")
			return
		}
		slog.Error("create github sync config failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create config")
		return
	}

	writeJSON(w, http.StatusCreated, syncConfigToResponse(config))
}

// UpdateGitHubSyncConfig updates an existing sync config.
func (h *Handler) UpdateGitHubSyncConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	configID := chi.URLParam(r, "id")

	var req struct {
		LabelFilter    *string `json:"label_filter"`
		DefaultAgentID *string `json:"default_agent_id"`
		GitHubToken    *string `json:"github_token"`
		Active         *bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params := db.UpdateGitHubSyncConfigParams{
		ID:          parseUUID(configID),
		WorkspaceID: parseUUID(workspaceID),
	}
	if req.LabelFilter != nil {
		params.LabelFilter = ptrToText(req.LabelFilter)
	}
	if req.DefaultAgentID != nil {
		params.DefaultAgentID = parseOptionalUUID(req.DefaultAgentID)
	}
	if req.GitHubToken != nil {
		params.GithubToken = ptrToText(req.GitHubToken)
	}
	if req.Active != nil {
		params.Active = ptrToBool(req.Active)
	}

	config, err := h.Queries.UpdateGitHubSyncConfig(r.Context(), params)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "config not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	writeJSON(w, http.StatusOK, syncConfigToResponse(config))
}

// DeleteGitHubSyncConfig deletes a sync config.
func (h *Handler) DeleteGitHubSyncConfig(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	configID := chi.URLParam(r, "id")

	err := h.Queries.DeleteGitHubSyncConfig(r.Context(), db.DeleteGitHubSyncConfigParams{
		ID:          parseUUID(configID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerGitHubSync manually triggers a sync for a specific config.
func (h *Handler) TriggerGitHubSync(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	configID := chi.URLParam(r, "id")

	config, err := h.Queries.GetGitHubSyncConfigInWorkspace(r.Context(), db.GetGitHubSyncConfigInWorkspaceParams{
		ID:          parseUUID(configID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "config not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	created, err := h.GitHubSyncService.SyncConfig(r.Context(), config)
	if err != nil {
		slog.Error("manual github sync failed", "config_id", configID, "error", err)
		writeError(w, http.StatusInternalServerError, "sync failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"created": created,
		"message": "sync completed",
	})
}

// parseOptionalUUID converts a *string to pgtype.UUID.
func parseOptionalUUID(s *string) pgtype.UUID {
	if s == nil || *s == "" {
		return pgtype.UUID{}
	}
	return parseUUID(*s)
}

// ptrToBool converts a *bool to pgtype.Bool.
func ptrToBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}
