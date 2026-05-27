"use client";

import { useState, useCallback } from "react";
import { GitBranch, Plus, Trash2, RefreshCw, Pencil } from "lucide-react";
import type { GitHubSyncConfig, CreateGitHubSyncConfigRequest, UpdateGitHubSyncConfigRequest } from "@multica/core/types";
import type { Agent } from "@multica/core/types";
import { Input } from "@multica/ui/components/ui/input";
import { Button } from "@multica/ui/components/ui/button";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import { Label } from "@multica/ui/components/ui/label";
import { Badge } from "@multica/ui/components/ui/badge";
import { Switch } from "@multica/ui/components/ui/switch";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@multica/ui/components/ui/select";
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
import { Tooltip, TooltipTrigger, TooltipContent } from "@multica/ui/components/ui/tooltip";
import { toast } from "sonner";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useWorkspaceId } from "@multica/core/hooks";
import { githubSyncListOptions, agentListOptions, workspaceKeys } from "@multica/core/workspace/queries";
import { api } from "@multica/core/api";
import { useAuthStore } from "@multica/core/auth";
import { memberListOptions } from "@multica/core/workspace/queries";

interface FormState {
  repo_owner: string;
  repo_name: string;
  label_filter: string;
  default_agent_id: string;
  github_token: string;
  active: boolean;
}

const emptyForm: FormState = {
  repo_owner: "",
  repo_name: "",
  label_filter: "multica",
  default_agent_id: "",
  github_token: "",
  active: true,
};

export function GitHubSyncTab() {
  const wsId = useWorkspaceId();
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const { data: configs = [], isLoading } = useQuery(githubSyncListOptions(wsId));
  const { data: agents = [] } = useQuery(agentListOptions(wsId));
  const { data: members = [] } = useQuery(memberListOptions(wsId));

  const activeAgents = agents.filter((a: Agent) => !a.archived_at);
  const currentMember = members.find((m) => m.user_id === user?.id);
  const canManage = currentMember?.role === "owner" || currentMember?.role === "admin";

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [syncing, setSyncing] = useState<string | null>(null);

  const openCreate = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setDialogOpen(true);
  }, []);

  const openEdit = useCallback((config: GitHubSyncConfig) => {
    setEditingId(config.id);
    setForm({
      repo_owner: config.repo_owner,
      repo_name: config.repo_name,
      label_filter: config.label_filter,
      default_agent_id: config.default_agent_id ?? "",
      github_token: "",
      active: config.active,
    });
    setDialogOpen(true);
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editingId) {
        const data: UpdateGitHubSyncConfigRequest = {
          label_filter: form.label_filter,
          default_agent_id: form.default_agent_id || undefined,
          active: form.active,
        };
        if (form.github_token) {
          data.github_token = form.github_token;
        }
        await api.updateGitHubSyncConfig(editingId, data);
        toast.success("Sync rule updated");
      } else {
        const data: CreateGitHubSyncConfigRequest = {
          repo_owner: form.repo_owner,
          repo_name: form.repo_name,
          label_filter: form.label_filter || "multica",
          default_agent_id: form.default_agent_id || undefined,
          github_token: form.github_token || undefined,
          active: form.active,
        };
        await api.createGitHubSyncConfig(data);
        toast.success("Sync rule created");
      }
      qc.invalidateQueries({ queryKey: workspaceKeys.githubSync(wsId) });
      setDialogOpen(false);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeleting(id);
    try {
      await api.deleteGitHubSyncConfig(id);
      qc.invalidateQueries({ queryKey: workspaceKeys.githubSync(wsId) });
      toast.success("Sync rule deleted");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to delete");
    } finally {
      setDeleting(null);
    }
  };

  const handleSync = async (id: string) => {
    setSyncing(id);
    try {
      const result = await api.triggerGitHubSync(id);
      qc.invalidateQueries({ queryKey: workspaceKeys.githubSync(wsId) });
      toast.success(`Sync completed: ${result.created} issue(s) created`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Sync failed");
    } finally {
      setSyncing(null);
    }
  };

  const agentName = (id: string | null) => {
    if (!id) return null;
    const agent = agents.find((a: Agent) => a.id === id);
    return agent?.name ?? null;
  };

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitBranch className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-semibold">GitHub Sync</h2>
          </div>
          {canManage && (
            <Button size="sm" onClick={openCreate}>
              <Plus className="h-3.5 w-3.5" />
              Add Sync Rule
            </Button>
          )}
        </div>

        <p className="text-xs text-muted-foreground">
          Sync GitHub issues with matching labels into Multica. Configure repository sync rules and assign a default agent.
        </p>

        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Card key={i}>
                <CardContent className="flex items-center gap-3">
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-48" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <Skeleton className="h-8 w-8 rounded" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : configs.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center text-sm text-muted-foreground">
              No sync rules configured yet.{canManage && " Click \"Add Sync Rule\" to get started."}
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-2">
            {configs.map((config) => (
              <Card key={config.id}>
                <CardContent className="flex items-center gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium truncate">
                        {config.repo_owner}/{config.repo_name}
                      </span>
                      <Badge variant={config.active ? "default" : "secondary"}>
                        {config.active ? "Active" : "Inactive"}
                      </Badge>
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5 space-x-2">
                      <span>Label: {config.label_filter}</span>
                      {agentName(config.default_agent_id) && (
                        <span>· Agent: {agentName(config.default_agent_id)}</span>
                      )}
                      {config.has_token && <span>· Token configured</span>}
                      {config.last_synced_at && (
                        <span>· Last sync: {new Date(config.last_synced_at).toLocaleString()}</span>
                      )}
                    </div>
                  </div>
                  {canManage && (
                    <div className="flex items-center gap-1">
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => handleSync(config.id)}
                              disabled={syncing === config.id}
                            >
                              <RefreshCw className={`h-3.5 w-3.5 ${syncing === config.id ? "animate-spin" : ""}`} />
                            </Button>
                          }
                        />
                        <TooltipContent>Trigger sync</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => openEdit(config)}
                            >
                              <Pencil className="h-3.5 w-3.5" />
                            </Button>
                          }
                        />
                        <TooltipContent>Edit</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => setDeleteConfirmId(config.id)}
                              disabled={deleting === config.id}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          }
                        />
                        <TooltipContent>Delete</TooltipContent>
                      </Tooltip>
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={(v) => { if (!v) setDialogOpen(false); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingId ? "Edit Sync Rule" : "Add Sync Rule"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="repo_owner">Repository Owner</Label>
                <Input
                  id="repo_owner"
                  value={form.repo_owner}
                  onChange={(e) => setForm((f) => ({ ...f, repo_owner: e.target.value }))}
                  placeholder="e.g. dmwork-org"
                  disabled={!!editingId}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="repo_name">Repository Name</Label>
                <Input
                  id="repo_name"
                  value={form.repo_name}
                  onChange={(e) => setForm((f) => ({ ...f, repo_name: e.target.value }))}
                  placeholder="e.g. dmworkim"
                  disabled={!!editingId}
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="label_filter">Label Filter</Label>
              <Input
                id="label_filter"
                value={form.label_filter}
                onChange={(e) => setForm((f) => ({ ...f, label_filter: e.target.value }))}
                placeholder="multica"
              />
              <p className="text-xs text-muted-foreground">
                Only GitHub issues with this label will be synced.
              </p>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="default_agent">Default Agent</Label>
              <Select
                value={form.default_agent_id || "__none__"}
                onValueChange={(v) => setForm((f) => ({ ...f, default_agent_id: !v || v === "__none__" ? "" : v }))}
              >
                <SelectTrigger size="sm" id="default_agent">
                  <SelectValue placeholder="None" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">None</SelectItem>
                  {activeAgents.map((agent: Agent) => (
                    <SelectItem key={agent.id} value={agent.id}>
                      {agent.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Synced issues will be assigned to this agent automatically.
              </p>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="github_token">GitHub Token</Label>
              <Input
                id="github_token"
                type="password"
                value={form.github_token}
                onChange={(e) => setForm((f) => ({ ...f, github_token: e.target.value }))}
                placeholder={editingId ? "Leave blank to keep existing" : "Optional — for private repos"}
              />
            </div>

            <div className="flex items-center gap-2">
              <Switch
                id="active"
                checked={form.active}
                onCheckedChange={(checked) => setForm((f) => ({ ...f, active: checked }))}
              />
              <Label htmlFor="active">Active</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>Cancel</Button>
            <Button
              onClick={handleSave}
              disabled={saving || (!editingId && (!form.repo_owner.trim() || !form.repo_name.trim()))}
            >
              {saving ? "Saving..." : editingId ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={!!deleteConfirmId} onOpenChange={(v) => { if (!v) setDeleteConfirmId(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete sync rule</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently remove the sync rule. Previously synced issues will not be affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={async () => {
                if (deleteConfirmId) await handleDelete(deleteConfirmId);
                setDeleteConfirmId(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
