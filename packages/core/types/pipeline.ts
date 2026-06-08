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

export interface ListPipelineTemplatesResponse {
  pipeline_templates: PipelineTemplate[];
  total: number;
}

export interface CreatePipelineTemplateRequest {
  name: string;
  description?: string | null;
  stages: PipelineStage[];
}

export interface UpdatePipelineTemplateRequest {
  name?: string | null;
  description?: string | null;
  stages?: PipelineStage[];
}

export interface IssuePipelineStatus {
  issue_id: string;
  pipeline_template_id: string | null;
  template_name?: string | null;
  current_stage: string | null;
  stages: PipelineStage[] | null;
  stage_results: Record<string, unknown> | null;
}

export interface AdvanceIssueStageRequest {
  result?: string;
  summary?: string;
}
