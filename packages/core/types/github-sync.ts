export interface GitHubSyncConfig {
  id: string;
  workspace_id: string;
  repo_owner: string;
  repo_name: string;
  label_filter: string;
  default_agent_id: string | null;
  has_token: boolean;
  active: boolean;
  last_synced_at: string | null;
  created_at: string;
}

export interface CreateGitHubSyncConfigRequest {
  repo_owner: string;
  repo_name: string;
  label_filter?: string;
  default_agent_id?: string;
  github_token?: string;
  active?: boolean;
}

export interface UpdateGitHubSyncConfigRequest {
  label_filter?: string;
  default_agent_id?: string;
  github_token?: string;
  active?: boolean;
}

export interface TriggerGitHubSyncResponse {
  created: number;
  message: string;
}
