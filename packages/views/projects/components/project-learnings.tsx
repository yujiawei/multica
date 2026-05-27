"use client";

import { useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Brain, ChevronRight, Plus, Trash2, X } from "lucide-react";
import { toast } from "sonner";
import type { LearningCategory } from "@multica/core/types";
import { projectLearningsOptions } from "@multica/core/learnings/queries";
import { useCreateLearning, useDeleteLearning } from "@multica/core/learnings/mutations";
import { useWorkspaceId } from "@multica/core/hooks";
import { Button } from "@multica/ui/components/ui/button";
import { cn } from "@multica/ui/lib/utils";

const CATEGORY_CONFIG: Record<LearningCategory, { label: string; color: string }> = {
  build: { label: "Build", color: "bg-blue-500/20 text-blue-700 dark:text-blue-400" },
  test: { label: "Test", color: "bg-green-500/20 text-green-700 dark:text-green-400" },
  pattern: { label: "Pattern", color: "bg-purple-500/20 text-purple-700 dark:text-purple-400" },
  error: { label: "Error", color: "bg-red-500/20 text-red-700 dark:text-red-400" },
  general: { label: "General", color: "bg-muted text-muted-foreground" },
};

const CATEGORIES: LearningCategory[] = ["build", "test", "pattern", "error", "general"];

export function ProjectLearnings({ projectId }: { projectId: string }) {
  const wsId = useWorkspaceId();
  const { data: learnings = [] } = useQuery(projectLearningsOptions(wsId, projectId));
  const createLearning = useCreateLearning(projectId);
  const deleteLearning = useDeleteLearning(projectId);

  const [open, setOpen] = useState(true);
  const [adding, setAdding] = useState(false);
  const [content, setContent] = useState("");
  const [category, setCategory] = useState<LearningCategory>("general");
  const [filter, setFilter] = useState<LearningCategory | null>(null);

  const filteredLearnings = filter ? learnings.filter((l) => l.category === filter) : learnings;

  const handleAdd = useCallback(() => {
    const trimmed = content.trim();
    if (!trimmed) return;
    createLearning.mutate(
      { content: trimmed, category },
      {
        onSuccess: () => {
          setContent("");
          setAdding(false);
          toast.success("Learning added");
        },
        onError: () => toast.error("Failed to add learning"),
      },
    );
  }, [content, category, createLearning]);

  const handleDelete = useCallback(
    (id: string) => {
      deleteLearning.mutate(id, {
        onError: () => toast.error("Failed to delete learning"),
      });
    },
    [deleteLearning],
  );

  return (
    <div>
      <div className="flex items-center justify-between">
        <button
          className={cn(
            "flex items-center gap-1 text-xs font-medium transition-colors",
            !open && "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => setOpen(!open)}
        >
          <ChevronRight
            className={cn(
              "h-3.5 w-3.5 shrink-0 text-muted-foreground transition-transform",
              open && "rotate-90",
            )}
          />
          <Brain className="h-3.5 w-3.5" />
          Learnings
          {learnings.length > 0 && (
            <span className="ml-1 text-muted-foreground">({learnings.length})</span>
          )}
        </button>
        {open && (
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground"
            onClick={() => setAdding(!adding)}
          >
            {adding ? <X className="h-3.5 w-3.5" /> : <Plus className="h-3.5 w-3.5" />}
          </Button>
        )}
      </div>

      {open && (
        <div className="mt-2 pl-2 space-y-2">
          {/* Category filter */}
          {learnings.length > 0 && (
            <div className="flex flex-wrap gap-1">
              <button
                className={cn(
                  "rounded-full px-2 py-0.5 text-[10px] transition-colors",
                  !filter
                    ? "bg-foreground/10 text-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )}
                onClick={() => setFilter(null)}
              >
                All
              </button>
              {CATEGORIES.map((c) => {
                const count = learnings.filter((l) => l.category === c).length;
                if (count === 0) return null;
                return (
                  <button
                    key={c}
                    className={cn(
                      "rounded-full px-2 py-0.5 text-[10px] transition-colors",
                      filter === c
                        ? CATEGORY_CONFIG[c].color
                        : "text-muted-foreground hover:text-foreground",
                    )}
                    onClick={() => setFilter(filter === c ? null : c)}
                  >
                    {CATEGORY_CONFIG[c].label}
                  </button>
                );
              })}
            </div>
          )}

          {/* Add form */}
          {adding && (
            <div className="space-y-2 rounded-md border p-2">
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="What did you learn?"
                className="w-full resize-none rounded-md bg-transparent text-xs placeholder:text-muted-foreground outline-none min-h-[60px]"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleAdd();
                }}
              />
              <div className="flex items-center justify-between">
                <div className="flex gap-1">
                  {CATEGORIES.map((c) => (
                    <button
                      key={c}
                      className={cn(
                        "rounded-full px-2 py-0.5 text-[10px] transition-colors",
                        category === c ? CATEGORY_CONFIG[c].color : "text-muted-foreground hover:text-foreground",
                      )}
                      onClick={() => setCategory(c)}
                    >
                      {CATEGORY_CONFIG[c].label}
                    </button>
                  ))}
                </div>
                <Button
                  size="sm"
                  className="h-6 text-xs"
                  onClick={handleAdd}
                  disabled={!content.trim() || createLearning.isPending}
                >
                  Add
                </Button>
              </div>
            </div>
          )}

          {/* Learnings list */}
          {filteredLearnings.length === 0 && !adding && (
            <p className="text-xs text-muted-foreground py-2">
              {learnings.length === 0
                ? "No learnings yet. Learnings are automatically captured from agent tasks."
                : "No learnings match this filter."}
            </p>
          )}
          <div className="space-y-1">
            {filteredLearnings.map((l) => (
              <div
                key={l.id}
                className="group flex items-start gap-2 rounded-md px-2 py-1.5 -mx-2 hover:bg-accent/50 transition-colors"
              >
                <span
                  className={cn(
                    "mt-0.5 shrink-0 rounded-full px-1.5 py-0.5 text-[10px] leading-none",
                    CATEGORY_CONFIG[l.category].color,
                  )}
                >
                  {CATEGORY_CONFIG[l.category].label}
                </span>
                <span className="flex-1 text-xs leading-relaxed">{l.content}</span>
                <button
                  className="shrink-0 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive transition-all"
                  onClick={() => handleDelete(l.id)}
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
