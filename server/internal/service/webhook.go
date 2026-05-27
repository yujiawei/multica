package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// WebhookService handles sending webhook notifications to external services.
type WebhookService struct {
	Queries *db.Queries
	client  *http.Client
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(q *db.Queries) *WebhookService {
	return &WebhookService{
		Queries: q,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WebhookPayload is the standard payload sent to webhook endpoints.
type WebhookPayload struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	Workspace WebhookWS      `json:"workspace"`
	Issue     *WebhookIssue  `json:"issue,omitempty"`
	Task      *WebhookTask   `json:"task,omitempty"`
	Result    *WebhookResult `json:"result,omitempty"`
}

type WebhookWS struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WebhookIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Status     string `json:"status"`
}

type WebhookTask struct {
	ID         string `json:"id"`
	Agent      string `json:"agent"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type WebhookResult struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
	PRUrl  string `json:"pr_url,omitempty"`
}

// SendEvent dispatches webhook notifications for a given event asynchronously.
func (s *WebhookService) SendEvent(ctx context.Context, workspaceID pgtype.UUID, event string, payload WebhookPayload) {
	webhooks, err := s.Queries.ListActiveWebhooksByWorkspaceAndEvent(ctx, db.ListActiveWebhooksByWorkspaceAndEventParams{
		WorkspaceID: workspaceID,
		Event:       event,
	})
	if err != nil {
		slog.Error("webhook: failed to list webhooks", "error", err, "event", event)
		return
	}
	if len(webhooks) == 0 {
		return
	}

	payload.Event = event
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: failed to marshal payload", "error", err, "event", event)
		return
	}

	for _, wh := range webhooks {
		go s.deliver(wh, body)
	}
}

// deliver sends the webhook HTTP POST with signature and retry.
func (s *WebhookService) deliver(wh db.Webhook, body []byte) {
	var signature string
	if wh.Secret.Valid && wh.Secret.String != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret.String))
		mac.Write(body)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest("POST", wh.Url, bytes.NewReader(body))
		if err != nil {
			slog.Error("webhook: failed to create request", "url", wh.Url, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if signature != "" {
			req.Header.Set("X-Webhook-Signature", signature)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			slog.Warn("webhook: delivery failed", "url", wh.Url, "attempt", attempt+1, "error", err)
			if attempt == 0 {
				time.Sleep(2 * time.Second)
				continue
			}
			return
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Debug("webhook: delivered", "url", wh.Url, "webhook_id", util.UUIDToString(wh.ID), "status", resp.StatusCode)
			return
		}

		slog.Warn("webhook: non-2xx response", "url", wh.Url, "attempt", attempt+1, "status", resp.StatusCode)
		if attempt == 0 {
			time.Sleep(2 * time.Second)
		}
	}
}

// SendTestEvent sends a test webhook payload to verify connectivity.
func (s *WebhookService) SendTestEvent(wh db.Webhook) error {
	payload := WebhookPayload{
		Event:     "webhook.test",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Workspace: WebhookWS{
			ID:   util.UUIDToString(wh.WorkspaceID),
			Name: "test",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", wh.Url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if wh.Secret.Valid && wh.Secret.String != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret.String))
		mac.Write(body)
		req.Header.Set("X-Webhook-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
