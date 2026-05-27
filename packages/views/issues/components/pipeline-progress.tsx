"use client";

import { memo } from "react";
import type { StageResult } from "@multica/core/types";
import { Workflow } from "lucide-react";
import { cn } from "@multica/ui/lib/utils";

export const PipelineProgress = memo(function PipelineProgress({
  currentStage,
  stageResults,
  className,
}: {
  currentStage: string;
  stageResults: Record<string, StageResult>;
  className?: string;
}) {
  const stageNames = Object.keys(stageResults);
  const completedCount = stageNames.filter(
    (name) => stageResults[name]?.completed_at,
  ).length;
  const total = stageNames.length;

  return (
    <div className={cn("mt-1.5 inline-flex items-center gap-1.5", className)}>
      <Workflow className="size-3 text-muted-foreground shrink-0" />
      <div className="flex items-center gap-0.5">
        {stageNames.map((name) => {
          const result = stageResults[name];
          const isCompleted = !!result?.completed_at;
          const isCurrent = name === currentStage;
          return (
            <div
              key={name}
              className={cn(
                "size-1.5 rounded-full",
                isCompleted
                  ? "bg-emerald-500"
                  : isCurrent
                    ? "bg-blue-500"
                    : "bg-muted-foreground/30",
              )}
            />
          );
        })}
      </div>
      <span className="text-[11px] text-muted-foreground tabular-nums font-medium">
        {completedCount}/{total}
      </span>
    </div>
  );
});
