package main

import (
	"context"
	"log/slog"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/util"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// registerGitHubSyncListeners registers event listeners for bidirectional
// GitHub issue sync. When a Multica issue transitions to "done", the
// corresponding GitHub issue is closed automatically.
func registerGitHubSyncListeners(bus *events.Bus, ghSync *service.GitHubSyncService) {
	bus.Subscribe(protocol.EventIssueUpdated, func(e events.Event) {
		payload, ok := e.Payload.(map[string]any)
		if !ok {
			return
		}

		statusChanged, _ := payload["status_changed"].(bool)
		if !statusChanged {
			return
		}

		issue, ok := payload["issue"].(handler.IssueResponse)
		if !ok {
			return
		}

		if issue.Status != "done" {
			return
		}

		go func() {
			ctx := context.Background()
			if err := ghSync.CloseGitHubIssue(ctx, util.ParseUUID(issue.ID)); err != nil {
				slog.Error("github sync: failed to close github issue",
					"multica_issue_id", issue.ID, "error", err)
			}
		}()
	})
}
