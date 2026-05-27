-- name: CreateWebhook :one
INSERT INTO webhook (workspace_id, url, secret, events, active)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetWebhook :one
SELECT * FROM webhook WHERE id = $1;

-- name: GetWebhookInWorkspace :one
SELECT * FROM webhook WHERE id = $1 AND workspace_id = $2;

-- name: ListWebhooksByWorkspace :many
SELECT * FROM webhook WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: ListActiveWebhooksByWorkspace :many
SELECT * FROM webhook WHERE workspace_id = $1 AND active = true;

-- name: ListActiveWebhooksByWorkspaceAndEvent :many
SELECT * FROM webhook
WHERE workspace_id = $1 AND active = true AND sqlc.arg('event')::text = ANY(events);

-- name: UpdateWebhook :one
UPDATE webhook SET
    url = COALESCE(sqlc.narg('url'), url),
    secret = COALESCE(sqlc.narg('secret'), secret),
    events = COALESCE(sqlc.narg('events'), events),
    active = COALESCE(sqlc.narg('active'), active),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhook WHERE id = $1;
