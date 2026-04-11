"use client";

import { useCallback, useEffect, useMemo } from "react";
import { toast } from "sonner";
import { ChevronRight, ListTodo } from "lucide-react";
import type { IssueStatus } from "@multica/core/types";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { useIssueViewStore, useClearFiltersOnWorkspaceChange } from "@multica/core/issues/stores/view-store";
import { useIssuesScopeStore } from "@multica/core/issues/stores/issues-scope-store";
import { ViewStoreProvider } from "@multica/core/issues/stores/view-store-context";
import { filterIssues } from "../utils/filter";
import { BOARD_STATUSES } from "@multica/core/issues/config";
import { useCurrentWorkspace } from "@multica/core/paths";
import { WorkspaceAvatar } from "../../workspace/workspace-avatar";
import { useWorkspaceId } from "@multica/core/hooks";
import { issueListOptions, childIssueProgressOptions } from "@multica/core/issues/queries";
import { useUpdateIssue } from "@multica/core/issues/mutations";
import { useAdvanceIssueStage } from "@multica/core/pipelines";
import { pipelineTemplateListOptions } from "@multica/core/pipelines";
import { useIssueSelectionStore } from "@multica/core/issues/stores/selection-store";
import { PageHeader } from "../../layout/page-header";
import { IssuesHeader } from "./issues-header";
import { BoardView } from "./board-view";
import { PipelineBoardView } from "./pipeline-board";
import { ListView } from "./list-view";
import { BatchActionToolbar } from "./batch-action-toolbar";

export function IssuesPage() {
  const wsId = useWorkspaceId();
  const { data: allIssues = [], isLoading: loading } = useQuery(issueListOptions(wsId));

  const workspace = useCurrentWorkspace();
  const scope = useIssuesScopeStore((s) => s.scope);
  const viewMode = useIssueViewStore((s) => s.viewMode);
  const boardGroupBy = useIssueViewStore((s) => s.boardGroupBy);
  const boardPipelineTemplateId = useIssueViewStore((s) => s.boardPipelineTemplateId);
  const statusFilters = useIssueViewStore((s) => s.statusFilters);
  const priorityFilters = useIssueViewStore((s) => s.priorityFilters);
  const assigneeFilters = useIssueViewStore((s) => s.assigneeFilters);
  const includeNoAssignee = useIssueViewStore((s) => s.includeNoAssignee);
  const creatorFilters = useIssueViewStore((s) => s.creatorFilters);
  const projectFilters = useIssueViewStore((s) => s.projectFilters);
  const includeNoProject = useIssueViewStore((s) => s.includeNoProject);

  // Clear filter state when switching between workspaces (URL-driven).
  useClearFiltersOnWorkspaceChange(useIssueViewStore, wsId);

  useEffect(() => {
    useIssueSelectionStore.getState().clear();
  }, [viewMode, scope]);

  // Scope pre-filter: narrow by assignee type
  const scopedIssues = useMemo(() => {
    if (scope === "members")
      return allIssues.filter((i) => i.assignee_type === "member");
    if (scope === "agents")
      return allIssues.filter((i) => i.assignee_type === "agent");
    return allIssues;
  }, [allIssues, scope]);

  const issues = useMemo(
    () => filterIssues(scopedIssues, { statusFilters, priorityFilters, assigneeFilters, includeNoAssignee, creatorFilters, projectFilters, includeNoProject }),
    [scopedIssues, statusFilters, priorityFilters, assigneeFilters, includeNoAssignee, creatorFilters, projectFilters, includeNoProject],
  );

  // Fetch sub-issue progress from the backend so counts are accurate
  // regardless of client-side pagination or filtering of done issues.
  const { data: childProgressMap = new Map() } = useQuery(childIssueProgressOptions(wsId));

  const visibleStatuses = useMemo(() => {
    if (statusFilters.length > 0)
      return BOARD_STATUSES.filter((s) => statusFilters.includes(s));
    return BOARD_STATUSES;
  }, [statusFilters]);

  const hiddenStatuses = useMemo(() => {
    return BOARD_STATUSES.filter((s) => !visibleStatuses.includes(s));
  }, [visibleStatuses]);

  // Pipeline templates for pipeline board view
  const { data: pipelineTemplates = [] } = useQuery(pipelineTemplateListOptions(wsId));

  // Auto-select pipeline template if not set or invalid
  const activePipelineTemplate = useMemo(() => {
    if (pipelineTemplates.length === 0) return null;
    const selected = pipelineTemplates.find((t) => t.id === boardPipelineTemplateId);
    if (selected) return selected;
    // Auto-select: prefer the template most issues are using
    const countByTemplate = new Map<string, number>();
    for (const issue of allIssues) {
      if (issue.pipeline_template_id) {
        countByTemplate.set(
          issue.pipeline_template_id,
          (countByTemplate.get(issue.pipeline_template_id) ?? 0) + 1,
        );
      }
    }
    let best = pipelineTemplates[0]!;
    let bestCount = 0;
    for (const t of pipelineTemplates) {
      const c = countByTemplate.get(t.id) ?? 0;
      if (c > bestCount) {
        best = t;
        bestCount = c;
      }
    }
    return best;
  }, [pipelineTemplates, boardPipelineTemplateId, allIssues]);

  // Auto-persist the selected template ID
  useEffect(() => {
    if (activePipelineTemplate && activePipelineTemplate.id !== boardPipelineTemplateId) {
      useIssueViewStore.getState().setBoardPipelineTemplateId(activePipelineTemplate.id);
    }
  }, [activePipelineTemplate, boardPipelineTemplateId]);

  const advanceStageMutation = useAdvanceIssueStage();
  const updateIssueMutation = useUpdateIssue();

  const handleAdvanceToStage = useCallback(
    (issueId: string, targetStage: string, newPosition?: number) => {
      const UNASSIGNED = "__pipeline_unassigned__";
      if (targetStage === UNASSIGNED) {
        // Cannot move issues out of a pipeline via drag
        return;
      }

      const issue = allIssues.find((i) => i.id === issueId);
      if (!issue) return;

      // If the issue is already at this stage, just reorder
      if (issue.current_stage === targetStage) {
        if (newPosition !== undefined) {
          updateIssueMutation.mutate(
            { id: issueId, position: newPosition },
            { onError: () => toast.error("Failed to move issue") },
          );
        }
        return;
      }

      // Use advanceIssueStage API to move to the target stage
      advanceStageMutation.mutate(
        { issueId },
        { onError: () => toast.error("Failed to advance stage") },
      );
    },
    [allIssues, advanceStageMutation, updateIssueMutation],
  );

  const handleMoveIssue = useCallback(
    (issueId: string, newStatus: IssueStatus, newPosition?: number) => {
      // Auto-switch to manual sort so drag ordering is preserved
      const viewState = useIssueViewStore.getState();
      if (viewState.sortBy !== "position") {
        viewState.setSortBy("position");
        viewState.setSortDirection("asc");
      }

      const updates: Partial<{ status: IssueStatus; position: number }> = {
        status: newStatus,
      };
      if (newPosition !== undefined) updates.position = newPosition;

      updateIssueMutation.mutate(
        { id: issueId, ...updates },
        { onError: () => toast.error("Failed to move issue") },
      );
    },
    [updateIssueMutation],
  );

  if (loading) {
    return (
      <div className="flex flex-1 min-h-0 flex-col">
        <div className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
          <Skeleton className="h-5 w-5 rounded" />
          <Skeleton className="h-4 w-32" />
        </div>
        <div className="flex h-12 shrink-0 items-center justify-between px-4">
          <div className="flex items-center gap-1">
            <Skeleton className="h-8 w-14 rounded-md" />
            <Skeleton className="h-8 w-20 rounded-md" />
            <Skeleton className="h-8 w-16 rounded-md" />
          </div>
          <div className="flex items-center gap-1">
            <Skeleton className="h-8 w-8 rounded-md" />
            <Skeleton className="h-8 w-8 rounded-md" />
            <Skeleton className="h-8 w-8 rounded-md" />
          </div>
        </div>
        {viewMode === "list" ? (
          <div className="flex-1 min-h-0 overflow-y-auto p-2 space-y-1">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="flex flex-1 min-h-0 gap-4 overflow-x-auto p-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex min-w-52 flex-1 flex-col gap-2">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-24 w-full rounded-lg" />
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      {/* Header 1: Workspace breadcrumb */}
      <PageHeader className="gap-1.5">
        <WorkspaceAvatar name={workspace?.name ?? "W"} size="sm" />
        <span className="text-sm text-muted-foreground">
          {workspace?.name ?? "Workspace"}
        </span>
        <ChevronRight className="h-3 w-3 text-muted-foreground" />
        <span className="text-sm font-medium">Issues</span>
      </PageHeader>

      <ViewStoreProvider store={useIssueViewStore}>
        {/* Header 2: Scope tabs + filters */}
        <IssuesHeader scopedIssues={scopedIssues} />

        {/* Content: scrollable */}
        {scopedIssues.length === 0 ? (
          <div className="flex flex-1 min-h-0 flex-col items-center justify-center gap-2 text-muted-foreground">
            <ListTodo className="h-10 w-10 text-muted-foreground/40" />
            <p className="text-sm">No issues yet</p>
            <p className="text-xs">Create an issue to get started.</p>
          </div>
        ) : (
          <div className="flex flex-col flex-1 min-h-0">
            {viewMode === "board" ? (
              <BoardView
                issues={issues}
                visibleStatuses={visibleStatuses}
                hiddenStatuses={hiddenStatuses}
                onMoveIssue={handleMoveIssue}
                childProgressMap={childProgressMap}
              />
            ) : (
              <ListView issues={issues} visibleStatuses={visibleStatuses} childProgressMap={childProgressMap} />
            )}
          </div>
        )}
        {viewMode === "list" && <BatchActionToolbar />}
      </ViewStoreProvider>
    </div>
  );
}
