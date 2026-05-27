package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type WebhookResponse struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	URL         string   `json:"url"`
	HasSecret   bool     `json:"has_secret"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func webhookToResponse(wh db.Webhook) WebhookResponse {
	return WebhookResponse{
		ID:          uuidToString(wh.ID),
		WorkspaceID: uuidToString(wh.WorkspaceID),
		URL:         wh.Url,
		HasSecret:   wh.Secret.Valid && wh.Secret.String != "",
		Events:      wh.Events,
		Active:      wh.Active,
		CreatedAt:   timestampToString(wh.CreatedAt),
		UpdatedAt:   timestampToString(wh.UpdatedAt),
	}
}

type CreateWebhookRequest struct {
	URL    string   `json:"url"`
	Secret *string  `json:"secret"`
	Events []string `json:"events"`
	Active *bool    `json:"active"`
}

type UpdateWebhookRequest struct {
	URL    *string  `json:"url"`
	Secret *string  `json:"secret"`
	Events []string `json:"events"`
	Active *bool    `json:"active"`
}

// ListWebhooks returns all webhooks for the current workspace.
func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.workspaceMember(w, r, workspaceID); !ok {
		return
	}

	webhooks, err := h.Queries.ListWebhooksByWorkspace(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list webhooks")
		return
	}

	resp := make([]WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		resp[i] = webhookToResponse(wh)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateWebhook creates a new webhook for the current workspace.
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	events := req.Events
	if len(events) == 0 {
		events = []string{"task.completed", "task.failed", "issue.status_changed"}
	}

	wh, err := h.Queries.CreateWebhook(r.Context(), db.CreateWebhookParams{
		WorkspaceID: parseUUID(workspaceID),
		Url:         req.URL,
		Secret:      ptrToText(req.Secret),
		Events:      events,
		Active:      active,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create webhook")
		return
	}

	writeJSON(w, http.StatusCreated, webhookToResponse(wh))
}

// UpdateWebhook updates an existing webhook.
func (h *Handler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	workspaceID := resolveWorkspaceID(r)

	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	if _, err := h.Queries.GetWebhookInWorkspace(r.Context(), db.GetWebhookInWorkspaceParams{
		ID:          parseUUID(webhookID),
		WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params := db.UpdateWebhookParams{
		ID: parseUUID(webhookID),
	}
	if req.URL != nil {
		params.Url = pgtype.Text{String: *req.URL, Valid: true}
	}
	if req.Secret != nil {
		params.Secret = pgtype.Text{String: *req.Secret, Valid: true}
	}
	if req.Events != nil {
		params.Events = req.Events
	}
	if req.Active != nil {
		params.Active = pgtype.Bool{Bool: *req.Active, Valid: true}
	}

	wh, err := h.Queries.UpdateWebhook(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update webhook")
		return
	}

	writeJSON(w, http.StatusOK, webhookToResponse(wh))
}

// DeleteWebhook deletes a webhook.
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	workspaceID := resolveWorkspaceID(r)

	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	if _, err := h.Queries.GetWebhookInWorkspace(r.Context(), db.GetWebhookInWorkspaceParams{
		ID:          parseUUID(webhookID),
		WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	if err := h.Queries.DeleteWebhook(r.Context(), parseUUID(webhookID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestWebhook sends a test payload to a webhook endpoint.
func (h *Handler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	workspaceID := resolveWorkspaceID(r)

	if _, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin"); !ok {
		return
	}

	wh, err := h.Queries.GetWebhookInWorkspace(r.Context(), db.GetWebhookInWorkspaceParams{
		ID:          parseUUID(webhookID),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	if err := h.WebhookService.SendTestEvent(wh); err != nil {
		writeError(w, http.StatusBadGateway, "webhook test failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
