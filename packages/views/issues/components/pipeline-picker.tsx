"use client";

import { Check, Workflow, X } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { pipelineTemplateListOptions } from "@multica/core/pipelines/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@multica/ui/components/ui/dropdown-menu";

export function PipelinePicker({
  pipelineTemplateId,
  onUpdate,
  triggerRender,
  align = "start",
}: {
  pipelineTemplateId: string | null;
  onUpdate: (templateId: string | null) => void;
  triggerRender?: React.ReactElement;
  align?: "start" | "center" | "end";
}) {
  const wsId = useWorkspaceId();
  const { data: templates = [] } = useQuery(pipelineTemplateListOptions(wsId));
  const current = templates.find((t) => t.id === pipelineTemplateId);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={triggerRender ? undefined : "flex items-center gap-1.5 cursor-pointer rounded px-1 -mx-1 hover:bg-accent/30 transition-colors overflow-hidden"}
        render={triggerRender}
      >
        <Workflow className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{current ? current.name : "No pipeline"}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-56">
        {templates.map((t) => (
          <DropdownMenuItem key={t.id} onClick={() => onUpdate(t.id)}>
            <span className="truncate">{t.name}</span>
            <span className="ml-auto text-[10px] text-muted-foreground tabular-nums">{t.stages.length} stages</span>
            {t.id === pipelineTemplateId && <Check className="ml-1 h-3.5 w-3.5 shrink-0" />}
          </DropdownMenuItem>
        ))}
        {templates.length > 0 && pipelineTemplateId && <DropdownMenuSeparator />}
        {pipelineTemplateId && (
          <DropdownMenuItem onClick={() => onUpdate(null)}>
            <X className="h-3.5 w-3.5 text-muted-foreground" />
            Remove pipeline
          </DropdownMenuItem>
        )}
        {templates.length === 0 && (
          <div className="px-2 py-1.5 text-xs text-muted-foreground">No pipeline templates</div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
