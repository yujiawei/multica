-- GitHub sync configuration per workspace + repo pair.
CREATE TABLE github_sync_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    label_filter TEXT NOT NULL DEFAULT 'multica',
    default_agent_id UUID REFERENCES agent(id) ON DELETE SET NULL,
    github_token TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, repo_owner, repo_name)
);

-- Mapping between GitHub issues and Multica issues.
CREATE TABLE github_issue_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES github_sync_config(id) ON DELETE CASCADE,
    github_repo TEXT NOT NULL,
    github_issue_number INTEGER NOT NULL,
    github_issue_url TEXT NOT NULL DEFAULT '',
    multica_issue_id UUID NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, github_repo, github_issue_number)
);

CREATE INDEX idx_github_issue_mapping_multica_issue ON github_issue_mapping(multica_issue_id);
