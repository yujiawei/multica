import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const pipelineKeys = {
  all: (wsId: string) => ["pipelines", wsId] as const,
  list: (wsId: string) => [...pipelineKeys.all(wsId), "list"] as const,
  detail: (wsId: string, id: string) =>
    [...pipelineKeys.all(wsId), "detail", id] as const,
  issueStatus: (wsId: string, issueId: string) =>
    [...pipelineKeys.all(wsId), "issue-status", issueId] as const,
};

export function pipelineTemplateListOptions(wsId: string) {
  return queryOptions({
    queryKey: pipelineKeys.list(wsId),
    queryFn: () => api.listPipelineTemplates(),
    select: (data) => data.pipeline_templates,
  });
}

export function pipelineTemplateDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: pipelineKeys.detail(wsId, id),
    queryFn: () => api.getPipelineTemplate(id),
  });
}

export function issuePipelineStatusOptions(wsId: string, issueId: string) {
  return queryOptions({
    queryKey: pipelineKeys.issueStatus(wsId, issueId),
    queryFn: () => api.getIssuePipelineStatus(issueId),
    enabled: !!issueId,
  });
}
