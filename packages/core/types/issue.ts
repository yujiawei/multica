export type IssueStatus =
  | "backlog"
  | "todo"
  | "in_progress"
  | "in_review"
  | "done"
  | "blocked"
  | "cancelled";

export type IssuePriority = "urgent" | "high" | "medium" | "low" | "none";

export type IssueAssigneeType = "member" | "agent";

export interface IssueReaction {
  id: string;
  issue_id: string;
  actor_type: string;
  actor_id: string;
  emoji: string;
  created_at: string;
}

export interface Issue {
  id: string;
  workspace_id: string;
  number: number;
  identifier: string;
  title: string;
  description: string | null;
  status: IssueStatus;
  priority: IssuePriority;
  assignee_type: IssueAssigneeType | null;
  assignee_id: string | null;
  creator_type: IssueAssigneeType;
  creator_id: string;
  parent_issue_id: string | null;
  project_id: string | null;
  position: number;
  due_date: string | null;
  pipeline_template_id: string | null;
  current_stage: string | null;
  stage_results?: Record<string, StageResult> | null;
  reactions?: IssueReaction[];
  created_at: string;
  updated_at: string;
}

export interface StageResult {
  started_at?: string;
  completed_at?: string;
  completed_by?: string;
  result?: string;
  summary?: string;
}

export interface PipelineStage {
  name: string;
  label: string;
  instructions: string;
}

export interface PipelineTemplate {
  id: string;
  workspace_id: string;
  name: string;
  description: string | null;
  stages: PipelineStage[];
  created_at: string;
  updated_at: string;
}

export interface CreatePipelineTemplateRequest {
  name: string;
  description?: string;
  stages: PipelineStage[];
}

export interface UpdatePipelineTemplateRequest {
  name?: string;
  description?: string;
  stages?: PipelineStage[];
}

export interface ListPipelineTemplatesResponse {
  pipeline_templates: PipelineTemplate[];
  total: number;
}

export interface PipelineStatusResponse {
  issue_id: string;
  pipeline_template_id: string | null;
  template_name: string | null;
  current_stage: string | null;
  stages: PipelineStage[] | null;
  stage_results: Record<string, StageResult> | null;
}
