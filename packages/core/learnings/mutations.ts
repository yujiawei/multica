import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { learningKeys } from "./queries";
import { useWorkspaceId } from "../hooks";
import type { CreateProjectLearningRequest, ListProjectLearningsResponse } from "../types";

export function useCreateLearning(projectId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreateProjectLearningRequest) =>
      api.createProjectLearning(projectId, data),
    onSuccess: (newLearning) => {
      qc.setQueryData<ListProjectLearningsResponse>(
        learningKeys.byProject(wsId, projectId),
        (old) =>
          old
            ? { ...old, learnings: [newLearning, ...old.learnings], total: old.total + 1 }
            : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: learningKeys.byProject(wsId, projectId) });
    },
  });
}

export function useDeleteLearning(projectId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (learningId: string) => api.deleteProjectLearning(learningId),
    onMutate: async (learningId) => {
      await qc.cancelQueries({ queryKey: learningKeys.byProject(wsId, projectId) });
      const prev = qc.getQueryData<ListProjectLearningsResponse>(
        learningKeys.byProject(wsId, projectId),
      );
      qc.setQueryData<ListProjectLearningsResponse>(
        learningKeys.byProject(wsId, projectId),
        (old) =>
          old
            ? { ...old, learnings: old.learnings.filter((l) => l.id !== learningId), total: old.total - 1 }
            : old,
      );
      return { prev };
    },
    onError: (_err, _id, ctx) => {
      if (ctx?.prev) qc.setQueryData(learningKeys.byProject(wsId, projectId), ctx.prev);
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: learningKeys.byProject(wsId, projectId) });
    },
  });
}
