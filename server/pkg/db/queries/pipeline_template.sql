-- name: ListPipelineTemplates :many
SELECT * FROM pipeline_template
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: GetPipelineTemplate :one
SELECT * FROM pipeline_template
WHERE id = $1;

-- name: GetPipelineTemplateInWorkspace :one
SELECT * FROM pipeline_template
WHERE id = $1 AND workspace_id = $2;

-- name: CreatePipelineTemplate :one
INSERT INTO pipeline_template (workspace_id, name, description, stages)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePipelineTemplate :one
UPDATE pipeline_template SET
    name = COALESCE(sqlc.narg('name'), name),
    description = sqlc.narg('description'),
    stages = COALESCE(sqlc.narg('stages'), stages),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePipelineTemplate :exec
DELETE FROM pipeline_template WHERE id = $1;

-- name: GetIssuePipelineState :one
SELECT id, pipeline_template_id, current_stage, stage_results
FROM issue
WHERE id = $1;

-- name: UpdateIssuePipeline :one
UPDATE issue SET
    pipeline_template_id = sqlc.narg('pipeline_template_id'),
    current_stage = sqlc.narg('current_stage'),
    stage_results = COALESCE(sqlc.narg('stage_results'), stage_results),
    updated_at = now()
WHERE id = $1
RETURNING *;
