"use client";

import { useState, useCallback } from "react";
import { Plus, Trash2, GripVertical, Pencil } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@multica/ui/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@multica/ui/components/ui/alert-dialog";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { toast } from "sonner";
import { useWorkspaceId } from "@multica/core/hooks";
import { pipelineTemplateListOptions } from "@multica/core/pipelines/queries";
import {
  useCreatePipelineTemplate,
  useUpdatePipelineTemplate,
  useDeletePipelineTemplate,
} from "@multica/core/pipelines/mutations";
import type { PipelineTemplate, PipelineStage } from "@multica/core/types";

const BUILTIN_TEMPLATES: { name: string; description: string; stages: PipelineStage[] }[] = [
  {
    name: "Bug Fix",
    description: "3-stage pipeline for bug fixes",
    stages: [
      { name: "investigate", label: "Investigate", instructions: "Reproduce and investigate the bug root cause" },
      { name: "fix", label: "Fix", instructions: "Implement the fix" },
      { name: "review", label: "Review", instructions: "Run /review and /codex for cross-review" },
    ],
  },
  {
    name: "Feature",
    description: "5-stage pipeline for feature development",
    stages: [
      { name: "plan", label: "Plan", instructions: "Run /office-hours to analyze requirements" },
      { name: "code", label: "Code", instructions: "Implement the feature" },
      { name: "review", label: "Self-Review", instructions: "Run /review" },
      { name: "cross_review", label: "Cross-Review", instructions: "Run /codex" },
      { name: "ship", label: "Ship", instructions: "Run /ship and create PR" },
    ],
  },
  {
    name: "Security Audit",
    description: "4-stage pipeline for security audits",
    stages: [
      { name: "scan", label: "Scan", instructions: "Run /cso for security audit" },
      { name: "fix", label: "Fix", instructions: "Fix identified vulnerabilities" },
      { name: "verify", label: "Verify", instructions: "Re-run /cso to verify fixes" },
      { name: "report", label: "Report", instructions: "Write audit report" },
    ],
  },
];

function StageEditor({
  stages,
  onChange,
}: {
  stages: PipelineStage[];
  onChange: (stages: PipelineStage[]) => void;
}) {
  const addStage = () => {
    onChange([...stages, { name: "", label: "", instructions: "" }]);
  };

  const removeStage = (index: number) => {
    onChange(stages.filter((_, i) => i !== index));
  };

  const updateStage = (index: number, field: keyof PipelineStage, value: string) => {
    const updated = stages.map((s, i) => {
      if (i !== index) return s;
      const newStage = { ...s, [field]: value };
      if (field === "label" && !s.name) {
        newStage.name = value.toLowerCase().replace(/\s+/g, "_").replace(/[^a-z0-9_]/g, "");
      }
      return newStage;
    });
    onChange(updated);
  };

  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">Stages</div>
      {stages.map((stage, i) => (
        <div key={i} className="flex items-start gap-2 p-3 border rounded-md bg-muted/30">
          <GripVertical className="size-4 mt-2.5 text-muted-foreground shrink-0" />
          <div className="flex-1 space-y-2">
            <div className="flex gap-2">
              <Input
                placeholder="Label"
                value={stage.label}
                onChange={(e) => updateStage(i, "label", e.target.value)}
                className="flex-1"
              />
              <Input
                placeholder="name"
                value={stage.name}
                onChange={(e) => updateStage(i, "name", e.target.value)}
                className="w-32 font-mono text-xs"
              />
            </div>
            <Textarea
              placeholder="Instructions for this stage..."
              value={stage.instructions}
              onChange={(e) => updateStage(i, "instructions", e.target.value)}
              rows={2}
              className="resize-none"
            />
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="shrink-0 mt-1"
            onClick={() => removeStage(i)}
          >
            <Trash2 className="size-4 text-muted-foreground" />
          </Button>
        </div>
      ))}
      <Button variant="outline" size="sm" onClick={addStage}>
        <Plus className="size-3.5 mr-1" />
        Add Stage
      </Button>
    </div>
  );
}

export function PipelinesTab() {
  const wsId = useWorkspaceId();
  const { data: templates, isLoading } = useQuery(pipelineTemplateListOptions(wsId));
  const createMutation = useCreatePipelineTemplate();
  const updateMutation = useUpdatePipelineTemplate();
  const deleteMutation = useDeletePipelineTemplate();

  const [editingTemplate, setEditingTemplate] = useState<PipelineTemplate | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formStages, setFormStages] = useState<PipelineStage[]>([]);

  const openCreate = useCallback((preset?: typeof BUILTIN_TEMPLATES[0]) => {
    setEditingTemplate(null);
    setFormName(preset?.name ?? "");
    setFormDescription(preset?.description ?? "");
    setFormStages(preset?.stages ?? [{ name: "", label: "", instructions: "" }]);
    setIsCreating(true);
  }, []);

  const openEdit = useCallback((tmpl: PipelineTemplate) => {
    setEditingTemplate(tmpl);
    setFormName(tmpl.name);
    setFormDescription(tmpl.description ?? "");
    setFormStages(tmpl.stages);
    setIsCreating(true);
  }, []);

  const handleSave = async () => {
    if (!formName.trim()) {
      toast.error("Name is required");
      return;
    }
    const validStages = formStages.filter((s) => s.name && s.label);
    if (validStages.length === 0) {
      toast.error("At least one stage with name and label is required");
      return;
    }
    try {
      if (editingTemplate) {
        await updateMutation.mutateAsync({
          id: editingTemplate.id,
          name: formName.trim(),
          description: formDescription.trim() || undefined,
          stages: validStages,
        });
        toast.success("Pipeline template updated");
      } else {
        await createMutation.mutateAsync({
          name: formName.trim(),
          description: formDescription.trim() || undefined,
          stages: validStages,
        });
        toast.success("Pipeline template created");
      }
      setIsCreating(false);
    } catch {
      toast.error("Failed to save pipeline template");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id);
      toast.success("Pipeline template deleted");
    } catch {
      toast.error("Failed to delete pipeline template");
    }
    setDeleteConfirmId(null);
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Pipeline Templates</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Configure multi-stage pipelines for issues. Each stage has independent instructions for agents.
        </p>
      </div>

      {/* Existing templates */}
      {isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      ) : templates && templates.length > 0 ? (
        <div className="space-y-3">
          {templates.map((tmpl) => (
            <Card key={tmpl.id}>
              <CardContent className="flex items-center justify-between py-4">
                <div className="min-w-0">
                  <div className="font-medium text-sm">{tmpl.name}</div>
                  {tmpl.description && (
                    <div className="text-xs text-muted-foreground mt-0.5">{tmpl.description}</div>
                  )}
                  <div className="flex items-center gap-1.5 mt-2 flex-wrap">
                    {tmpl.stages.map((stage, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 text-[11px] font-medium rounded-full bg-muted text-muted-foreground"
                      >
                        {stage.label}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-1 shrink-0 ml-4">
                  <Button variant="ghost" size="icon" onClick={() => openEdit(tmpl)}>
                    <Pencil className="size-4" />
                  </Button>
                  <Button variant="ghost" size="icon" onClick={() => setDeleteConfirmId(tmpl.id)}>
                    <Trash2 className="size-4 text-muted-foreground" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="text-center py-8 text-sm text-muted-foreground border rounded-lg">
          No pipeline templates yet. Create one or use a built-in template.
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 flex-wrap">
        <Button size="sm" onClick={() => openCreate()}>
          <Plus className="size-3.5 mr-1" />
          Create Template
        </Button>
        <span className="text-xs text-muted-foreground">or use a built-in:</span>
        {BUILTIN_TEMPLATES.map((preset) => (
          <Button
            key={preset.name}
            variant="outline"
            size="sm"
            onClick={() => openCreate(preset)}
          >
            {preset.name}
          </Button>
        ))}
      </div>

      {/* Create/Edit Dialog */}
      <Dialog open={isCreating} onOpenChange={setIsCreating}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editingTemplate ? "Edit Pipeline Template" : "Create Pipeline Template"}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">Name</label>
              <Input
                placeholder="e.g., Feature Development"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Description</label>
              <Input
                placeholder="Optional description"
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
              />
            </div>
            <StageEditor stages={formStages} onChange={setFormStages} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsCreating(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSave}
              disabled={createMutation.isPending || updateMutation.isPending}
            >
              {createMutation.isPending || updateMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={!!deleteConfirmId} onOpenChange={() => setDeleteConfirmId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete pipeline template?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the template. Issues already using this pipeline will keep their current stage data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirmId && handleDelete(deleteConfirmId)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
