# .pi /issue → iw Core / Codex 对齐审计（进行中）

基线：`rontian/.pi@78ce0c8`、`rontian/issue-workflow@40f98c1`、`rontian/codex-workflow@0e670c5`（2026-10-29）。

本文件是 **差异记录，不是完成证明**。参考 `.pi/workflows/github-issue/{LIFECYCLE,STATE,STATE_GUARDS,SETTLEMENT,GOAL,ACTIONS}.md` 和源码；`.pi` 是需求参考，最终协议仍以本仓 Core 的 Agent-neutral 语义为准。

| 原有能力 | 当前 Core | 缺口/归属 | 验收条件 |
| --- | --- | --- | --- |
| 九态 lifecycle、暂停、等待、延期、阻塞 | v2 lifecycle / events | wait 需要多项外部条件模型 | 多条件 recheck 不允许简单 `--resolved` 越权跳过 |
| Task Contract + 计划 Txx/checklist 同步 | v1 contract + cross-Issue `tasks/next` | **Core 缺失 Issue 内稳定 task/phase/current/next 状态与严格推进门禁** | 最后未完成 Txx 不允许 complete/defer；checkpoint/context 显示当前任务 |
| Settlement 逐项 decision + accepted risk | 无与 `.pi` 等价的结构化 item/fact/disposition，只有 validation evidence | **Core 缺失 settlement engine**，交互收集属 Host | 仅 owner-decidable manual/real-resource/acceptance 可 accepted risk；永不伪造 pass；HEAD 变化失效；defer 清理当前 segment 决定 |
| defer release | 目前只有事件迁移与原因/下一步；本 PR 增加 Git 发布和 review guard | 仍缺 settled current-segment/task/settlement gate | 不能凭 `--reason` 认定工作结束；Issue 保持 OPEN |
| completion gate | 结构化 validation/review、Git drift + optional review policy | 未核对全部 acceptance checklist、残余风险与 GitHub close policy；本 PR 修复空 HEAD validation / stale passed review | 自动门禁 fail closed；GitHub close 独立显式策略 |
| portable handoff/restore | 已实现；有 fake Git 单测 | 缺双工作目录真实 GitHub 黑盒 | exact published HEAD、clean worktree、restore conflict/fail closed |
| v1 迁移、event replay/idempotency | 已实现并有测试 | 仍缺真实 GitHub failure injection | fork/duplicate/unknown outcome 不产生双事件 |
| Pi Todo/Goal/settlement UI、continuous driver | 不在 Core | **Codex / Pi 各自 Host Adapter** | host runtime 只能作为 projection；不能产生新 canonical fact |
| Codex machine adapter | `codex-workflow` 仍引用 v1 schema | **Core 验收完成后升级 Adapter** | manifest、Skill、smoke 改用 v2；端到端 Codex session 测试 |

## 本 PR 已做的有限防护

- validation 记录在检查 current HEAD 时不得因 `observed_head` 缺失而被当作有效 PASS；
- 有过 passed review 时，若 HEAD 已变化，则 completion 不再允许沿用旧 review；
- defer 前检查 worktree clean、没有 Git drift、review 无遗留问题、项目 review policy、确切 published remote HEAD。

这些防护不能证明 `defer` 已达到 `.pi` 的完整 Settlement readiness；**不能用它们作为启动 Codex v2 正式适配的依据**。未完成项仍须由 `issue-workflow#3` 追踪并在真实仓黑盒验收后关闭。

## 正确推进顺序

1. Core：定义稳定 task/settlement/wait-item schema、CLI 和 fail-closed gates，以及上下文投影。
2. Core：`pi-test` 真实仓黑盒；双工作目录 portable restore；迁移、review、Git drift、contract drift、operation retry。
3. Core：冻结 v2 文档、fixtures，并关闭 #3。
4. Codex：更新 manifest / SKILL / conformance smoke，验证多会话 Codex 接力；不复制 Go Core。
5. Pi-specific Goal/Todo/UI 仅供后续 `pi-workflow` 实现参考，不导入 Core。
