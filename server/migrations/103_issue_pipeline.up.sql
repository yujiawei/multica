-- Pipeline template
CREATE TABLE pipeline_template (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    stages JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipeline_template_workspace ON pipeline_template(workspace_id);

-- Issue pipeline state
ALTER TABLE issue ADD COLUMN pipeline_template_id UUID REFERENCES pipeline_template(id) ON DELETE SET NULL;
ALTER TABLE issue ADD COLUMN current_stage TEXT;
ALTER TABLE issue ADD COLUMN stage_results JSONB DEFAULT '{}';
