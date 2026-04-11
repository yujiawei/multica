# Multica Fork 改造计划

Fork: https://github.com/yujiawei/multica
Upstream: https://github.com/multica-ai/multica
设计灵感: garrytan/gstack (Pipeline-as-Code + Self-Learning)

## P0 — Webhook 通知引擎

Issue/Task 状态变更时触发 HTTP POST，推送到 DMWork/Discord/Slack。

- 新增 `webhook` 表（workspace_id, url, events[], secret, active）
- `TaskService` 在 task complete/fail 时调用
- Payload: `{event, issue, task, pr_url, agent, timestamp}`
- Workspace Settings UI 增加 Webhook 配置页

### 改动文件
- `server/migrations/100_webhooks.sql` — 新表
- `server/pkg/db/queries/webhook.sql` — sqlc 查询
- `server/internal/handler/webhook.go` — CRUD API
- `server/internal/service/webhook.go` — 发送逻辑
- `server/internal/service/task.go` — 完成时触发
- `server/cmd/server/router.go` — 路由注册
- `apps/web/features/webhooks/` — 前端配置页

## P0 — GitHub Issue 双向同步

- GitHub Issue 打 `multica` label → 自动创建 Multica Issue
- Multica done → PR closes #N 自动关闭 GitHub Issue
- Workspace Settings 配置 repo + label 映射
- 支持 cron polling 或 GitHub Webhook

### 改动文件
- `server/migrations/101_github_sync.sql` — 配置表
- `server/internal/service/github_sync.go` — 同步逻辑
- `server/internal/handler/github_webhook.go` — Webhook 接收
- `server/cmd/server/router.go` — 路由

## P1 — Project Learnings（项目记忆）

Agent 执行后自动记录 learnings，下次同项目自动注入。

### 改动文件
- `server/migrations/102_project_learnings.sql`
- `server/pkg/db/queries/learning.sql`
- `server/internal/daemon/prompt.go` — 注入 learnings
- `server/internal/daemon/learning.go` — 提取+存储
- `server/cmd/multica/cmd_learn.go` — CLI

## P2 — Issue Pipeline

Issue 支持多 Stage，看板 Stage 子列。

## P2 — Codex Sandbox 配置化

`MULTICA_CODEX_SANDBOX` 环境变量。

## 技术约束

- Migration 编号从 100 开始（upstream 当前到 039）
- 新功能 feature flag 控制
- 保持 upstream merge 兼容
