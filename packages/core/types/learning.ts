export type LearningCategory = "build" | "test" | "pattern" | "error" | "general";

export interface ProjectLearning {
  id: string;
  workspace_id: string;
  project_id: string | null;
  content: string;
  source: string | null;
  source_task_id: string | null;
  category: LearningCategory;
  created_at: string;
}

export interface CreateProjectLearningRequest {
  content: string;
  source?: string;
  source_task_id?: string;
  category?: LearningCategory;
}

export interface ListProjectLearningsResponse {
  learnings: ProjectLearning[];
  total: number;
}
