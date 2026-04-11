"use client";

import { useState, useCallback, useMemo, useEffect, useRef } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  pointerWithin,
  closestCenter,
  type CollisionDetection,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from "@dnd-kit/core";
import { arrayMove, SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useDroppable } from "@dnd-kit/core";
import { Workflow } from "lucide-react";
import type { Issue, PipelineStage } from "@multica/core/types";
import { useViewStore } from "@multica/core/issues/stores/view-store-context";
import type { SortField, SortDirection } from "@multica/core/issues/stores/view-store";
import { sortIssues } from "../utils/sort";
import { BoardCardContent, DraggableBoardCard } from "./board-card";
import type { ChildProgress } from "./list-row";

const UNASSIGNED_COLUMN_ID = "__pipeline_unassigned__";

/** Build column ID arrays for pipeline stage grouping. */
function buildPipelineColumns(
  issues: Issue[],
  stages: PipelineStage[],
  pipelineTemplateId: string,
  sortBy: SortField,
  sortDirection: SortDirection,
): Record<string, string[]> {
  const cols: Record<string, string[]> = {};
  for (const stage of stages) {
    const stageIssues = issues.filter(
      (i) =>
        i.pipeline_template_id === pipelineTemplateId &&
        i.current_stage === stage.name,
    );
    cols[stage.name] = sortIssues(stageIssues, sortBy, sortDirection).map(
      (i) => i.id,
    );
  }
  // Unassigned: issues not on this pipeline
  const unassigned = issues.filter(
    (i) => i.pipeline_template_id !== pipelineTemplateId,
  );
  cols[UNASSIGNED_COLUMN_ID] = sortIssues(unassigned, sortBy, sortDirection).map(
    (i) => i.id,
  );
  return cols;
}

function computePosition(
  ids: string[],
  activeId: string,
  issueMap: Map<string, Issue>,
): number {
  const idx = ids.indexOf(activeId);
  if (idx === -1) return 0;
  const getPos = (id: string) => issueMap.get(id)?.position ?? 0;
  if (ids.length === 1) return issueMap.get(activeId)?.position ?? 0;
  if (idx === 0) return getPos(ids[1]!) - 1;
  if (idx === ids.length - 1) return getPos(ids[idx - 1]!) + 1;
  return (getPos(ids[idx - 1]!) + getPos(ids[idx + 1]!)) / 2;
}

function findColumn(
  columns: Record<string, string[]>,
  id: string,
  columnIds: string[],
): string | null {
  if (columnIds.includes(id)) return id;
  for (const [colId, ids] of Object.entries(columns)) {
    if (ids.includes(id)) return colId;
  }
  return null;
}

const EMPTY_PROGRESS_MAP = new Map<string, ChildProgress>();

export function PipelineBoardView({
  issues,
  stages,
  pipelineTemplateId,
  onAdvanceToStage,
  childProgressMap = EMPTY_PROGRESS_MAP,
}: {
  issues: Issue[];
  stages: PipelineStage[];
  pipelineTemplateId: string;
  onAdvanceToStage: (
    issueId: string,
    targetStage: string,
    newPosition?: number,
  ) => void;
  childProgressMap?: Map<string, ChildProgress>;
}) {
  const sortBy = useViewStore((s) => s.sortBy);
  const sortDirection = useViewStore((s) => s.sortDirection);

  const columnIds = useMemo(
    () => [...stages.map((s) => s.name), UNASSIGNED_COLUMN_ID],
    [stages],
  );
  const COLUMN_IDS_SET = useMemo(() => new Set(columnIds), [columnIds]);

  const kanbanCollision: CollisionDetection = useCallback(
    (args) => {
      const pointer = pointerWithin(args);
      if (pointer.length > 0) {
        const cards = pointer.filter((c) => !COLUMN_IDS_SET.has(c.id as string));
        if (cards.length > 0) return cards;
      }
      return closestCenter(args);
    },
    [COLUMN_IDS_SET],
  );

  // Drag state
  const [activeIssue, setActiveIssue] = useState<Issue | null>(null);
  const isDraggingRef = useRef(false);

  const [columns, setColumns] = useState<Record<string, string[]>>(() =>
    buildPipelineColumns(issues, stages, pipelineTemplateId, sortBy, sortDirection),
  );
  const columnsRef = useRef(columns);
  columnsRef.current = columns;

  useEffect(() => {
    if (!isDraggingRef.current) {
      setColumns(
        buildPipelineColumns(issues, stages, pipelineTemplateId, sortBy, sortDirection),
      );
    }
  }, [issues, stages, pipelineTemplateId, sortBy, sortDirection]);

  const recentlyMovedRef = useRef(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => {
      recentlyMovedRef.current = false;
    });
    return () => cancelAnimationFrame(id);
  }, [columns]);

  const issueMap = useMemo(() => {
    const map = new Map<string, Issue>();
    for (const issue of issues) map.set(issue.id, issue);
    return map;
  }, [issues]);

  const issueMapRef = useRef(issueMap);
  if (!isDraggingRef.current) {
    issueMapRef.current = issueMap;
  }

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    isDraggingRef.current = true;
    const issue = issueMapRef.current.get(event.active.id as string) ?? null;
    setActiveIssue(issue);
  }, []);

  const handleDragOver = useCallback(
    (event: DragOverEvent) => {
      const { active, over } = event;
      if (!over || recentlyMovedRef.current) return;

      const activeId = active.id as string;
      const overId = over.id as string;

      setColumns((prev) => {
        const activeCol = findColumn(prev, activeId, columnIds);
        const overCol = findColumn(prev, overId, columnIds);
        if (!activeCol || !overCol || activeCol === overCol) return prev;

        recentlyMovedRef.current = true;
        const oldIds = prev[activeCol]!.filter((id) => id !== activeId);
        const newIds = [...prev[overCol]!];
        const overIndex = newIds.indexOf(overId);
        const insertIndex = overIndex >= 0 ? overIndex : newIds.length;
        newIds.splice(insertIndex, 0, activeId);
        return { ...prev, [activeCol]: oldIds, [overCol]: newIds };
      });
    },
    [columnIds],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      isDraggingRef.current = false;
      setActiveIssue(null);

      const resetColumns = () =>
        setColumns(
          buildPipelineColumns(issues, stages, pipelineTemplateId, sortBy, sortDirection),
        );

      if (!over) {
        resetColumns();
        return;
      }

      const activeId = active.id as string;
      const overId = over.id as string;

      const cols = columnsRef.current;
      const activeCol = findColumn(cols, activeId, columnIds);
      const overCol = findColumn(cols, overId, columnIds);
      if (!activeCol || !overCol) {
        resetColumns();
        return;
      }

      // Same-column reorder
      let finalColumns = cols;
      if (activeCol === overCol) {
        const ids = cols[activeCol]!;
        const oldIndex = ids.indexOf(activeId);
        const newIndex = ids.indexOf(overId);
        if (oldIndex !== -1 && newIndex !== -1 && oldIndex !== newIndex) {
          const reordered = arrayMove(ids, oldIndex, newIndex);
          finalColumns = { ...cols, [activeCol]: reordered };
          setColumns(finalColumns);
        }
      }

      const finalCol = findColumn(finalColumns, activeId, columnIds);
      if (!finalCol) {
        resetColumns();
        return;
      }

      const map = issueMapRef.current;
      const finalIds = finalColumns[finalCol]!;
      const newPosition = computePosition(finalIds, activeId, map);

      onAdvanceToStage(activeId, finalCol, newPosition);
    },
    [issues, stages, pipelineTemplateId, sortBy, sortDirection, columnIds, onAdvanceToStage],
  );

  const stageLabels = useMemo(() => {
    const map = new Map<string, string>();
    for (const stage of stages) {
      map.set(stage.name, stage.label);
    }
    map.set(UNASSIGNED_COLUMN_ID, "Unassigned");
    return map;
  }, [stages]);

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={kanbanCollision}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex flex-1 min-h-0 gap-4 overflow-x-auto p-4">
        {columnIds.map((colId) => (
          <PipelineColumn
            key={colId}
            columnId={colId}
            label={stageLabels.get(colId) ?? colId}
            isUnassigned={colId === UNASSIGNED_COLUMN_ID}
            issueIds={columns[colId] ?? []}
            issueMap={issueMapRef.current}
            childProgressMap={childProgressMap}
          />
        ))}
      </div>

      <DragOverlay dropAnimation={null}>
        {activeIssue ? (
          <div className="w-[280px] rotate-2 scale-105 cursor-grabbing opacity-90 shadow-lg shadow-black/10">
            <BoardCardContent
              issue={activeIssue}
              childProgress={childProgressMap.get(activeIssue.id)}
            />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}

function PipelineColumn({
  columnId,
  label,
  isUnassigned,
  issueIds,
  issueMap,
  childProgressMap,
}: {
  columnId: string;
  label: string;
  isUnassigned: boolean;
  issueIds: string[];
  issueMap: Map<string, Issue>;
  childProgressMap?: Map<string, ChildProgress>;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: columnId });

  const resolvedIssues = useMemo(
    () =>
      issueIds.flatMap((id) => {
        const issue = issueMap.get(id);
        return issue ? [issue] : [];
      }),
    [issueIds, issueMap],
  );

  return (
    <div
      className={`flex w-[280px] shrink-0 flex-col rounded-xl p-2 ${
        isUnassigned ? "bg-muted/40" : "bg-violet-500/5"
      }`}
    >
      <div className="mb-2 flex items-center justify-between px-1.5">
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 rounded px-2 py-0.5 text-xs font-semibold ${
              isUnassigned
                ? "bg-muted text-muted-foreground"
                : "bg-violet-500/15 text-violet-700 dark:text-violet-300"
            }`}
          >
            <Workflow className="h-3 w-3" />
            {label}
          </span>
          <span className="text-xs text-muted-foreground">{issueIds.length}</span>
        </div>
      </div>
      <div
        ref={setNodeRef}
        className={`min-h-[200px] flex-1 space-y-2 overflow-y-auto rounded-lg p-1 transition-colors ${
          isOver ? "bg-accent/60" : ""
        }`}
      >
        <SortableContext items={issueIds} strategy={verticalListSortingStrategy}>
          {resolvedIssues.map((issue) => (
            <DraggableBoardCard
              key={issue.id}
              issue={issue}
              childProgress={childProgressMap?.get(issue.id)}
            />
          ))}
        </SortableContext>
        {issueIds.length === 0 && (
          <p className="py-8 text-center text-xs text-muted-foreground">
            No issues
          </p>
        )}
      </div>
    </div>
  );
}
