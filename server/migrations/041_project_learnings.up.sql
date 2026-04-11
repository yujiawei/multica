CREATE TABLE project_learning (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    project_id UUID REFERENCES project(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    source TEXT,
    source_task_id UUID,
    category TEXT NOT NULL DEFAULT 'general'
        CHECK (category IN ('build', 'test', 'pattern', 'error', 'general')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learning_workspace_project ON project_learning(workspace_id, project_id);
CREATE INDEX idx_learning_source_task ON project_learning(source_task_id);
