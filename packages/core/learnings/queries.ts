import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const learningKeys = {
  all: (wsId: string) => ["learnings", wsId] as const,
  byProject: (wsId: string, projectId: string) =>
    [...learningKeys.all(wsId), "project", projectId] as const,
};

export function projectLearningsOptions(wsId: string, projectId: string) {
  return queryOptions({
    queryKey: learningKeys.byProject(wsId, projectId),
    queryFn: () => api.listProjectLearnings(projectId),
    select: (data) => data.learnings,
  });
}
