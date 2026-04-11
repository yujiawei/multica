package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type ProjectLearningResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	ProjectID    *string `json:"project_id"`
	Content      string  `json:"content"`
	Source       *string `json:"source"`
	SourceTaskID *string `json:"source_task_id"`
	Category     string  `json:"category"`
	CreatedAt    string  `json:"created_at"`
}

func learningToResponse(l db.ProjectLearning) ProjectLearningResponse {
	return ProjectLearningResponse{
		ID:           uuidToString(l.ID),
		WorkspaceID:  uuidToString(l.WorkspaceID),
		ProjectID:    uuidToPtr(l.ProjectID),
		Content:      l.Content,
		Source:       textToPtr(l.Source),
		SourceTaskID: uuidToPtr(l.SourceTaskID),
		Category:     l.Category,
		CreatedAt:    timestampToString(l.CreatedAt),
	}
}

type CreateProjectLearningRequest struct {
	Content      string  `json:"content"`
	Source       *string `json:"source"`
	SourceTaskID *string `json:"source_task_id"`
	Category     string  `json:"category"`
}

func (h *Handler) ListProjectLearnings(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := resolveWorkspaceID(r)

	var categoryFilter pgtype.Text
	if c := r.URL.Query().Get("category"); c != "" {
		categoryFilter = pgtype.Text{String: c, Valid: true}
	}

	learnings, err := h.Queries.ListProjectLearnings(r.Context(), db.ListProjectLearningsParams{
		WorkspaceID: parseUUID(workspaceID),
		ProjectID:   parseUUID(projectID),
		Category:    categoryFilter,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list learnings")
		return
	}

	resp := make([]ProjectLearningResponse, len(learnings))
	for i, l := range learnings {
		resp[i] = learningToResponse(l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"learnings": resp, "total": len(resp)})
}

func (h *Handler) CreateProjectLearning(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	workspaceID := resolveWorkspaceID(r)

	// Verify project exists in workspace.
	if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
		ID: parseUUID(projectID), WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	actorType, actorID := h.resolveActor(r, userID, workspaceID)

	var req CreateProjectLearningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	category := req.Category
	if category == "" {
		category = "general"
	}

	var sourceTaskID pgtype.UUID
	if req.SourceTaskID != nil {
		sourceTaskID = parseUUID(*req.SourceTaskID)
	}

	learning, err := h.Queries.CreateProjectLearning(r.Context(), db.CreateProjectLearningParams{
		WorkspaceID:  parseUUID(workspaceID),
		ProjectID:    parseUUID(projectID),
		Content:      req.Content,
		Source:       ptrToText(req.Source),
		SourceTaskID: sourceTaskID,
		Category:     category,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create learning")
		return
	}

	resp := learningToResponse(learning)
	h.publish(protocol.EventLearningCreated, workspaceID, actorType, actorID, map[string]any{"learning": resp})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) DeleteProjectLearning(w http.ResponseWriter, r *http.Request) {
	learningID := chi.URLParam(r, "learningId")
	workspaceID := resolveWorkspaceID(r)

	if _, err := h.Queries.GetProjectLearningInWorkspace(r.Context(), db.GetProjectLearningInWorkspaceParams{
		ID: parseUUID(learningID), WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "learning not found")
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if err := h.Queries.DeleteProjectLearning(r.Context(), parseUUID(learningID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete learning")
		return
	}

	h.publish(protocol.EventLearningDeleted, workspaceID, "member", userID, map[string]any{"learning_id": learningID})
	w.WriteHeader(http.StatusNoContent)
}

// GetLearningsForInjection returns learnings for a project, intended for daemon injection.
func (h *Handler) GetLearningsForInjection(w http.ResponseWriter, r *http.Request) {
	workspaceID := resolveWorkspaceID(r)
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	learnings, err := h.Queries.ListLearningsByProject(r.Context(), db.ListLearningsByProjectParams{
		WorkspaceID: parseUUID(workspaceID),
		ProjectID:   parseUUID(projectID),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list learnings")
		return
	}

	resp := make([]ProjectLearningResponse, len(learnings))
	for i, l := range learnings {
		resp[i] = learningToResponse(l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"learnings": resp})
}
