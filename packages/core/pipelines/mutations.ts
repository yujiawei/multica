import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { pipelineKeys } from "./queries";
import { useWorkspaceId } from "../hooks";
import type { ListPipelineTemplatesResponse, CreatePipelineTemplateRequest, UpdatePipelineTemplateRequest } from "../types";

export function useCreatePipelineTemplate() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreatePipelineTemplateRequest) => api.createPipelineTemplate(data),
    onSuccess: (newTemplate) => {
      qc.setQueryData<ListPipelineTemplatesResponse>(pipelineKeys.list(wsId), (old) =>
        old && !old.pipeline_templates.some((t) => t.id === newTemplate.id)
          ? { ...old, pipeline_templates: [...old.pipeline_templates, newTemplate], total: old.total + 1 }
          : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useUpdatePipelineTemplate() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdatePipelineTemplateRequest) =>
      api.updatePipelineTemplate(id, data),
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: pipelineKeys.detail(wsId, vars.id) });
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useDeletePipelineTemplate() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.deletePipelineTemplate(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: pipelineKeys.list(wsId) });
      const prevList = qc.getQueryData<ListPipelineTemplatesResponse>(pipelineKeys.list(wsId));
      qc.setQueryData<ListPipelineTemplatesResponse>(pipelineKeys.list(wsId), (old) =>
        old ? { ...old, pipeline_templates: old.pipeline_templates.filter((t) => t.id !== id), total: old.total - 1 } : old,
      );
      return { prevList };
    },
    onError: (_err, _id, ctx) => {
      if (ctx?.prevList) qc.setQueryData(pipelineKeys.list(wsId), ctx.prevList);
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useAdvanceIssueStage() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ issueId, ...data }: { issueId: string; result?: string; summary?: string }) =>
      api.advanceIssueStage(issueId, data),
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: pipelineKeys.issueStatus(wsId, vars.issueId) });
      qc.invalidateQueries({ queryKey: ["issues", wsId] });
    },
  });
}
