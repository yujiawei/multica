# Cancelling running tasks

When an agent task is misbehaving — emitting tool calls in a tight loop,
pushing more commits than you wanted, or stuck on a slow turn — you can
cancel it from the CLI.

## Single task

```bash
# By full task UUID
multica issue cancel-task 9ee2c778-1234-5678-90ab-cdef01234567

# By short ID prefix (the form shown by `issue runs`)
multica issue cancel-task 9ee2c778

# Scoped to a specific issue when the prefix is ambiguous
multica issue cancel-task 9ee2c778 --issue YUJ-513

# JSON output for scripting
multica issue cancel-task <id> --output json
```

The command calls `POST /api/tasks/{taskId}/cancel`. The server flips
the task row to `status=cancelled` and broadcasts a `task:cancelled`
event. Combined with the daemon-side change in
[#2107](https://github.com/multica-ai/multica/pull/2107) this also
triggers the local agent to stop emitting tool calls promptly instead
of waiting for its own internal timeout.

## All running tasks for an issue

There is no `cancel-tasks-for-issue` subcommand yet, but the same
effect is two pipes away:

```bash
ISSUE=YUJ-513
multica issue runs $ISSUE --output json \
  | jq -r '.[] | select(.status == "running" or .status == "queued") | .id' \
  | xargs -I{} multica issue cancel-task {} --issue $ISSUE
```

This is appropriate for runaway agents that have queued multiple
follow-up tasks.

## When to use this

- An agent has crossed its boundary and is iterating on something it
  should not (e.g. piling up `t.Skip` markers, polishing past the
  scope you asked for).
- Codex / Claude / Gemini turns are stuck and you want to abort
  rather than wait for the provider timeout.
- You changed your mind about an issue and want to free up the
  runtime slot immediately, without waiting for the heartbeat /
  garbage collector.

## What it does NOT do

- It does **not** rerun the task. Use `multica issue rerun <issue>`
  for that.
- It does **not** unassign the issue. Cancel-task only stops the
  in-flight or queued execution; the agent assignment stays so a
  subsequent rerun goes to the same agent.
- It does **not** undo any side effects the agent already produced
  (commits already pushed stay pushed, comments stay posted, etc.).
  Cancellation is a forward-only stop, not a rollback.

## Troubleshooting

If `cancel-task` returns 404 it means the task ID does not belong to
the current workspace. Re-run `multica issue runs <issue>` to confirm
you have the right short ID and try again.
