-- name: ListProjectLearnings :many
SELECT * FROM project_learning
WHERE workspace_id = $1
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
ORDER BY created_at DESC;

-- name: GetProjectLearning :one
SELECT * FROM project_learning
WHERE id = $1;

-- name: GetProjectLearningInWorkspace :one
SELECT * FROM project_learning
WHERE id = $1 AND workspace_id = $2;

-- name: CreateProjectLearning :one
INSERT INTO project_learning (
    workspace_id, project_id, content, source, source_task_id, category
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: DeleteProjectLearning :exec
DELETE FROM project_learning WHERE id = $1;

-- name: ListLearningsByProject :many
SELECT * FROM project_learning
WHERE workspace_id = $1 AND project_id = $2
ORDER BY created_at DESC;

-- name: CountLearningsByProject :one
SELECT count(*) FROM project_learning
WHERE workspace_id = $1 AND project_id = $2;
