"use client";

import { Workflow, Check, Clock, Circle } from "lucide-react";
import type { StageResult } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";

function formatDuration(startedAt: string, completedAt: string): string {
  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();
  const diffMs = end - start;
  if (diffMs < 60_000) return `${Math.round(diffMs / 1000)}s`;
  if (diffMs < 3_600_000) return `${Math.round(diffMs / 60_000)}m`;
  return `${Math.round(diffMs / 3_600_000)}h`;
}

function shortTime(iso: string): string {
  return new Date(iso).toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
  });
}

export function PipelineTimeline({
  currentStage,
  stageResults,
}: {
  currentStage: string | null;
  stageResults: Record<string, StageResult>;
}) {
  const stageNames = Object.keys(stageResults);

  return (
    <div>
      <div className="text-xs font-medium mb-2 flex items-center gap-1">
        <Workflow className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        Pipeline
      </div>
      <div className="pl-2 space-y-0">
        {stageNames.map((name, i) => {
          const result = stageResults[name];
          const isCompleted = !!result?.completed_at;
          const isCurrent = name === currentStage;
          const isLast = i === stageNames.length - 1;

          return (
            <div key={name} className="flex items-start gap-2">
              {/* Timeline line + dot */}
              <div className="flex flex-col items-center shrink-0">
                <div
                  className={cn(
                    "flex items-center justify-center size-5 rounded-full",
                    isCompleted
                      ? "bg-emerald-500/15 text-emerald-500"
                      : isCurrent
                        ? "bg-blue-500/15 text-blue-500"
                        : "bg-muted text-muted-foreground",
                  )}
                >
                  {isCompleted ? (
                    <Check className="size-3" />
                  ) : isCurrent ? (
                    <Clock className="size-3" />
                  ) : (
                    <Circle className="size-2.5" />
                  )}
                </div>
                {!isLast && (
                  <div className={cn("w-px h-4", isCompleted ? "bg-emerald-500/30" : "bg-border")} />
                )}
              </div>
              {/* Label and info */}
              <div className="min-w-0 pb-2">
                <div className={cn("text-xs font-medium", isCurrent && "text-blue-500")}>
                  {name}
                </div>
                {result?.started_at && (
                  <div className="text-[11px] text-muted-foreground">
                    {isCompleted && result.completed_at
                      ? `${shortTime(result.started_at)} — ${formatDuration(result.started_at, result.completed_at)}`
                      : `Started ${shortTime(result.started_at)}`}
                  </div>
                )}
                {result?.summary && (
                  <div className="text-[11px] text-muted-foreground mt-0.5 line-clamp-2">
                    {result.summary}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
