-- name: CreateGitHubSyncConfig :one
INSERT INTO github_sync_config (
    workspace_id, repo_owner, repo_name, label_filter, default_agent_id, github_token, active
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetGitHubSyncConfig :one
SELECT * FROM github_sync_config
WHERE id = $1;

-- name: GetGitHubSyncConfigInWorkspace :one
SELECT * FROM github_sync_config
WHERE id = $1 AND workspace_id = $2;

-- name: ListGitHubSyncConfigs :many
SELECT * FROM github_sync_config
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: ListActiveGitHubSyncConfigs :many
SELECT * FROM github_sync_config
WHERE active = true
ORDER BY last_synced_at ASC NULLS FIRST;

-- name: UpdateGitHubSyncConfigLastSynced :exec
UPDATE github_sync_config SET last_synced_at = now() WHERE id = $1;

-- name: UpdateGitHubSyncConfig :one
UPDATE github_sync_config SET
    label_filter = COALESCE(sqlc.narg('label_filter'), label_filter),
    default_agent_id = sqlc.narg('default_agent_id'),
    github_token = COALESCE(sqlc.narg('github_token'), github_token),
    active = COALESCE(sqlc.narg('active'), active)
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteGitHubSyncConfig :exec
DELETE FROM github_sync_config WHERE id = $1 AND workspace_id = $2;

-- name: CreateGitHubIssueMapping :one
INSERT INTO github_issue_mapping (
    workspace_id, config_id, github_repo, github_issue_number, github_issue_url, multica_issue_id
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetGitHubIssueMappingByGitHub :one
SELECT * FROM github_issue_mapping
WHERE workspace_id = $1 AND github_repo = $2 AND github_issue_number = $3;

-- name: GetGitHubIssueMappingByMulticaIssue :one
SELECT * FROM github_issue_mapping
WHERE multica_issue_id = $1;

-- name: ListGitHubIssueMappingsByConfig :many
SELECT * FROM github_issue_mapping
WHERE config_id = $1
ORDER BY synced_at DESC;
