"use client";

import { useState } from "react";
import { Webhook, Plus, Trash2, Pencil, Play, Globe } from "lucide-react";
import type {
  Webhook as WebhookType,
  CreateWebhookRequest,
  UpdateWebhookRequest,
} from "@multica/core/types";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import { Button } from "@multica/ui/components/ui/button";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import { Badge } from "@multica/ui/components/ui/badge";
import { Switch } from "@multica/ui/components/ui/switch";
import { Checkbox } from "@multica/ui/components/ui/checkbox";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@multica/ui/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@multica/ui/components/ui/alert-dialog";
import { Tooltip, TooltipTrigger, TooltipContent } from "@multica/ui/components/ui/tooltip";
import { toast } from "sonner";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useWorkspaceId } from "@multica/core/hooks";
import { webhookListOptions, workspaceKeys } from "@multica/core/workspace/queries";
import { api } from "@multica/core/api";

const AVAILABLE_EVENTS = [
  { value: "issue.created", label: "Issue Created" },
  { value: "issue.assigned", label: "Issue Assigned" },
  { value: "issue.status_changed", label: "Issue Status Changed" },
  { value: "task.completed", label: "Task Completed" },
  { value: "task.failed", label: "Task Failed" },
] as const;

function WebhookForm({
  initial,
  onSubmit,
  onCancel,
  submitting,
}: {
  initial?: WebhookType;
  onSubmit: (data: CreateWebhookRequest | UpdateWebhookRequest) => void;
  onCancel: () => void;
  submitting: boolean;
}) {
  const [url, setUrl] = useState(initial?.url ?? "");
  const [secret, setSecret] = useState("");
  const [events, setEvents] = useState<string[]>(
    initial?.events ?? ["task.completed", "task.failed", "issue.status_changed"],
  );
  const [active, setActive] = useState(initial?.active ?? true);

  const isEdit = !!initial;
  const urlValid = url.startsWith("https://");
  const canSubmit = urlValid && events.length > 0 && !submitting;

  const toggleEvent = (value: string) => {
    setEvents((prev) =>
      prev.includes(value) ? prev.filter((e) => e !== value) : [...prev, value],
    );
  };

  const handleSubmit = () => {
    if (!canSubmit) return;
    if (isEdit) {
      const data: UpdateWebhookRequest = { url, events, active };
      if (secret) data.secret = secret;
      onSubmit(data);
    } else {
      const data: CreateWebhookRequest = { url, events, active };
      if (secret) data.secret = secret;
      onSubmit(data);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="webhook-url">URL</Label>
        <Input
          id="webhook-url"
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/webhook"
        />
        {url && !urlValid && (
          <p className="text-xs text-destructive">URL must start with https://</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="webhook-secret">
          Secret {isEdit && <span className="text-muted-foreground">(leave empty to keep current)</span>}
        </Label>
        <Input
          id="webhook-secret"
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          placeholder={isEdit ? "••••••••" : "Optional signing secret"}
        />
      </div>

      <div className="space-y-2">
        <Label>Events</Label>
        <div className="space-y-2">
          {AVAILABLE_EVENTS.map((evt) => (
            <label key={evt.value} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={events.includes(evt.value)}
                onCheckedChange={() => toggleEvent(evt.value)}
              />
              {evt.label}
              <code className="text-xs text-muted-foreground">{evt.value}</code>
            </label>
          ))}
        </div>
        {events.length === 0 && (
          <p className="text-xs text-destructive">Select at least one event</p>
        )}
      </div>

      <div className="flex items-center gap-2">
        <Switch checked={active} onCheckedChange={setActive} id="webhook-active" />
        <Label htmlFor="webhook-active">Active</Label>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onCancel} disabled={submitting}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} disabled={!canSubmit}>
          {submitting ? (isEdit ? "Saving..." : "Creating...") : isEdit ? "Save" : "Create"}
        </Button>
      </DialogFooter>
    </div>
  );
}

export function WebhooksTab() {
  const wsId = useWorkspaceId();
  const qc = useQueryClient();
  const { data: webhooks = [], isLoading } = useQuery(webhookListOptions(wsId));

  const [formOpen, setFormOpen] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<WebhookType | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  const invalidate = () => qc.invalidateQueries({ queryKey: workspaceKeys.webhooks(wsId) });

  const handleCreate = async (data: CreateWebhookRequest | UpdateWebhookRequest) => {
    setSubmitting(true);
    try {
      await api.createWebhook(data as CreateWebhookRequest);
      setFormOpen(false);
      invalidate();
      toast.success("Webhook created");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to create webhook");
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpdate = async (data: CreateWebhookRequest | UpdateWebhookRequest) => {
    if (!editingWebhook) return;
    setSubmitting(true);
    try {
      await api.updateWebhook(editingWebhook.id, data as UpdateWebhookRequest);
      setEditingWebhook(null);
      invalidate();
      toast.success("Webhook updated");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to update webhook");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteWebhook(id);
      invalidate();
      toast.success("Webhook deleted");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to delete webhook");
    }
  };

  const handleToggle = async (wh: WebhookType) => {
    setTogglingId(wh.id);
    try {
      await api.updateWebhook(wh.id, { active: !wh.active });
      invalidate();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to toggle webhook");
    } finally {
      setTogglingId(null);
    }
  };

  const handleTest = async (id: string) => {
    setTestingId(id);
    try {
      await api.testWebhook(id);
      toast.success("Test payload sent");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Test failed");
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-semibold">Webhooks</h2>
          </div>
          <Button size="sm" onClick={() => setFormOpen(true)}>
            <Plus className="h-4 w-4" />
            Add Webhook
          </Button>
        </div>

        <p className="text-xs text-muted-foreground">
          Webhooks notify external services when events occur in your workspace.
        </p>

        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Card key={i}>
                <CardContent className="flex items-center gap-3">
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-48" />
                    <Skeleton className="h-3 w-64" />
                  </div>
                  <Skeleton className="h-8 w-8 rounded" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : webhooks.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <Webhook className="h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm text-muted-foreground">No webhooks configured</p>
              <p className="text-xs text-muted-foreground mt-1">
                Add a webhook to receive event notifications
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-2">
            {webhooks.map((wh) => (
              <Card key={wh.id}>
                <CardContent className="flex items-center gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium truncate">{wh.url}</span>
                      <Badge variant={wh.active ? "default" : "secondary"}>
                        {wh.active ? "Active" : "Inactive"}
                      </Badge>
                      {wh.has_secret && (
                        <Badge variant="outline">Signed</Badge>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      {wh.events.join(", ")} · Created {new Date(wh.created_at).toLocaleDateString()}
                    </div>
                  </div>

                  <div className="flex items-center gap-1 shrink-0">
                    <Switch
                      checked={wh.active}
                      onCheckedChange={() => handleToggle(wh)}
                      disabled={togglingId === wh.id}
                      aria-label={wh.active ? "Disable" : "Enable"}
                    />

                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => handleTest(wh.id)}
                            disabled={testingId === wh.id}
                            aria-label="Test webhook"
                          >
                            <Play className="h-3.5 w-3.5" />
                          </Button>
                        }
                      />
                      <TooltipContent>Send test</TooltipContent>
                    </Tooltip>

                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => setEditingWebhook(wh)}
                            aria-label="Edit webhook"
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
                            onClick={() => setDeleteConfirmId(wh.id)}
                            aria-label="Delete webhook"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        }
                      />
                      <TooltipContent>Delete</TooltipContent>
                    </Tooltip>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* Create dialog */}
      <Dialog open={formOpen} onOpenChange={(v) => { if (!v) setFormOpen(false); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Webhook</DialogTitle>
          </DialogHeader>
          <WebhookForm
            onSubmit={handleCreate}
            onCancel={() => setFormOpen(false)}
            submitting={submitting}
          />
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog open={!!editingWebhook} onOpenChange={(v) => { if (!v) setEditingWebhook(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Webhook</DialogTitle>
          </DialogHeader>
          {editingWebhook && (
            <WebhookForm
              initial={editingWebhook}
              onSubmit={handleUpdate}
              onCancel={() => setEditingWebhook(null)}
              submitting={submitting}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <AlertDialog open={!!deleteConfirmId} onOpenChange={(v) => { if (!v) setDeleteConfirmId(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete webhook</AlertDialogTitle>
            <AlertDialogDescription>
              This webhook will be permanently deleted and will no longer receive events. This cannot be undone.
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
