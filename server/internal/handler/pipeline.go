package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// PipelineStage represents a single stage in a pipeline template.
type PipelineStage struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	Instructions string `json:"instructions"`
}

// PipelineTemplateResponse is the JSON response for a pipeline template.
type PipelineTemplateResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Stages      json.RawMessage `json:"stages"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

func pipelineTemplateToResponse(t db.PipelineTemplate) PipelineTemplateResponse {
	return PipelineTemplateResponse{
		ID:          uuidToString(t.ID),
		WorkspaceID: uuidToString(t.WorkspaceID),
		Name:        t.Name,
		Description: textToPtr(t.Description),
		Stages:      json.RawMessage(t.Stages),
		CreatedAt:   timestampToString(t.CreatedAt),
		UpdatedAt:   timestampToString(t.UpdatedAt),
	}
}

type CreatePipelineTemplateRequest struct {
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Stages      json.RawMessage `json:"stages"`
}

type UpdatePipelineTemplateRequest struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Stages      json.RawMessage `json:"stages"`
}

func (h *Handler) ListPipelineTemplates(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	templates, err := h.Queries.ListPipelineTemplates(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipeline templates")
		return
	}
	resp := make([]PipelineTemplateResponse, len(templates))
	for i, t := range templates {
		resp[i] = pipelineTemplateToResponse(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pipeline_templates": resp, "total": len(resp)})
}

func (h *Handler) GetPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	tmpl, err := h.Queries.GetPipelineTemplateInWorkspace(r.Context(), db.GetPipelineTemplateInWorkspaceParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(workspaceID),
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline template not found")
		return
	}
	writeJSON(w, http.StatusOK, pipelineTemplateToResponse(tmpl))
}

func (h *Handler) CreatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreatePipelineTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Stages) == 0 {
		writeError(w, http.StatusBadRequest, "stages is required")
		return
	}
	// Validate stages JSON
	var stages []PipelineStage
	if err := json.Unmarshal(req.Stages, &stages); err != nil {
		writeError(w, http.StatusBadRequest, "invalid stages format")
		return
	}
	if len(stages) == 0 {
		writeError(w, http.StatusBadRequest, "at least one stage is required")
		return
	}

	workspaceID := h.resolveWorkspaceID(r)
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	tmpl, err := h.Queries.CreatePipelineTemplate(r.Context(), db.CreatePipelineTemplateParams{
		WorkspaceID: parseUUID(workspaceID),
		Name:        req.Name,
		Description: ptrToText(req.Description),
		Stages:      req.Stages,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pipeline template")
		return
	}
	resp := pipelineTemplateToResponse(tmpl)
	h.publish(protocol.EventPipelineTemplateCreated, workspaceID, "member", userID, map[string]any{"pipeline_template": resp})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) UpdatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetPipelineTemplateInWorkspace(r.Context(), db.GetPipelineTemplateInWorkspaceParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "pipeline template not found")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req UpdatePipelineTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Validate stages if provided
	if len(req.Stages) > 0 {
		var stages []PipelineStage
		if err := json.Unmarshal(req.Stages, &stages); err != nil {
			writeError(w, http.StatusBadRequest, "invalid stages format")
			return
		}
		if len(stages) == 0 {
			writeError(w, http.StatusBadRequest, "at least one stage is required")
			return
		}
	}
	params := db.UpdatePipelineTemplateParams{
		ID: parseUUID(id),
	}
	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	} else {
		// Allow clearing description via null
		params.Description = pgtype.Text{Valid: false}
	}
	if len(req.Stages) > 0 {
		params.Stages = req.Stages
	}
	tmpl, err := h.Queries.UpdatePipelineTemplate(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update pipeline template")
		return
	}
	resp := pipelineTemplateToResponse(tmpl)
	h.publish(protocol.EventPipelineTemplateUpdated, workspaceID, "member", userID, map[string]any{"pipeline_template": resp})
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeletePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	if _, err := h.Queries.GetPipelineTemplateInWorkspace(r.Context(), db.GetPipelineTemplateInWorkspaceParams{
		ID:          parseUUID(id),
		WorkspaceID: parseUUID(workspaceID),
	}); err != nil {
		writeError(w, http.StatusNotFound, "pipeline template not found")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if err := h.Queries.DeletePipelineTemplate(r.Context(), parseUUID(id)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete pipeline template")
		return
	}
	h.publish(protocol.EventPipelineTemplateDeleted, workspaceID, "member", userID, map[string]any{"pipeline_template_id": id})
	w.WriteHeader(http.StatusNoContent)
}

// PipelineStatusResponse is the JSON response for an issue's pipeline status.
type PipelineStatusResponse struct {
	IssueID            string          `json:"issue_id"`
	PipelineTemplateID *string         `json:"pipeline_template_id"`
	TemplateName       *string         `json:"template_name"`
	CurrentStage       *string         `json:"current_stage"`
	Stages             json.RawMessage `json:"stages"`
	StageResults       json.RawMessage `json:"stage_results"`
}

func (h *Handler) GetIssuePipelineStatus(w http.ResponseWriter, r *http.Request) {
	issueID := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, issueID)
	if !ok {
		return
	}

	resp := PipelineStatusResponse{
		IssueID:            uuidToString(issue.ID),
		PipelineTemplateID: uuidToPtr(issue.PipelineTemplateID),
		CurrentStage:       textToPtr(issue.CurrentStage),
		StageResults:       normalizeStageResults(issue.StageResults),
	}

	if issue.PipelineTemplateID.Valid {
		tmpl, err := h.Queries.GetPipelineTemplate(r.Context(), issue.PipelineTemplateID)
		if err == nil {
			name := tmpl.Name
			resp.TemplateName = &name
			resp.Stages = json.RawMessage(tmpl.Stages)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

type AdvanceStageRequest struct {
	Result  *string         `json:"result"`
	Summary *string         `json:"summary"`
}

func (h *Handler) AdvanceIssueStage(w http.ResponseWriter, r *http.Request) {
	issueID := chi.URLParam(r, "id")
	issue, ok := h.loadIssueForUser(w, r, issueID)
	if !ok {
		return
	}

	if !issue.PipelineTemplateID.Valid {
		writeError(w, http.StatusBadRequest, "issue has no pipeline")
		return
	}

	tmpl, err := h.Queries.GetPipelineTemplate(r.Context(), issue.PipelineTemplateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load pipeline template")
		return
	}

	var stages []PipelineStage
	if err := json.Unmarshal(tmpl.Stages, &stages); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid pipeline stages")
		return
	}
	if len(stages) == 0 {
		writeError(w, http.StatusBadRequest, "pipeline has no stages")
		return
	}

	var req AdvanceStageRequest
	json.NewDecoder(r.Body).Decode(&req)

	userID := requestUserID(r)
	workspaceID := h.resolveWorkspaceID(r)
	actorType, actorID := h.resolveActor(r, userID, workspaceID)

	// Find current stage index
	currentIdx := -1
	currentStageName := ""
	if issue.CurrentStage.Valid {
		currentStageName = issue.CurrentStage.String
		for i, s := range stages {
			if s.Name == currentStageName {
				currentIdx = i
				break
			}
		}
	}

	// Update stage results for the current stage
	stageResults := make(map[string]any)
	if len(issue.StageResults) > 0 {
		json.Unmarshal(issue.StageResults, &stageResults)
	}

	if currentStageName != "" {
		stageResult := map[string]any{
			"completed_at": time.Now().UTC().Format(time.RFC3339),
			"completed_by": actorID,
		}
		if req.Result != nil {
			stageResult["result"] = *req.Result
		}
		if req.Summary != nil {
			stageResult["summary"] = *req.Summary
		}
		stageResults[currentStageName] = stageResult
	}

	stageResultsJSON, _ := json.Marshal(stageResults)

	// Determine next stage
	nextIdx := currentIdx + 1
	var nextStage pgtype.Text
	issueStatus := issue.Status

	if nextIdx < len(stages) {
		nextStage = pgtype.Text{String: stages[nextIdx].Name, Valid: true}
		// Record start time for the next stage
		stageResults[stages[nextIdx].Name] = map[string]any{
			"started_at": time.Now().UTC().Format(time.RFC3339),
		}
		stageResultsJSON, _ = json.Marshal(stageResults)
	} else {
		// Last stage completed — mark issue as done
		issueStatus = "done"
	}

	// Update the issue pipeline state
	updated, err := h.Queries.UpdateIssuePipeline(r.Context(), db.UpdateIssuePipelineParams{
		ID:                 issue.ID,
		PipelineTemplateID: issue.PipelineTemplateID,
		CurrentStage:       nextStage,
		StageResults:       stageResultsJSON,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to advance stage")
		return
	}

	// Update issue status if pipeline completed
	if issueStatus != issue.Status {
		updated, err = h.Queries.UpdateIssueStatus(r.Context(), db.UpdateIssueStatusParams{
			ID:     issue.ID,
			Status: issueStatus,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update issue status")
			return
		}
	}

	issuePrefix := h.getIssuePrefix(r.Context(), issue.WorkspaceID)
	resp := issueToResponse(updated, issuePrefix)
	h.publish(protocol.EventIssueUpdated, workspaceID, actorType, actorID, map[string]any{"issue": resp})
	writeJSON(w, http.StatusOK, resp)
}
