# .pi /issue → iw Core / Codex 对齐审计（进行中）

基线：`rontian/.pi@78ce0c8`、`rontian/issue-workflow@40f98c1`、`rontian/codex-workflow@0e670c5`（2026-10-29）。

本文件是 **差异记录，不是完成证明**。参考 `.pi/workflows/github-issue/{LIFECYCLE,STATE,STATE_GUARDS,SETTLEMENT,GOAL,ACTIONS}.md` 和源码；`.pi` 是需求参考，最终协议仍以本仓 Core 的 Agent-neutral 语义为准。

| 原有能力 | 当前 Core | 缺口/归属 | 验收条件 |
| --- | --- | --- | --- |
| 九态 lifecycle、暂停、等待、延期、阻塞 | v2 lifecycle / events + 本 PR 命名 external_wait 条件 | 多条件已支持全量 recheck；仍需真实 GitHub 黑盒、和 Settlement 的衔接 | 多条件 recheck 不允许简单 `--resolved` 越权跳过 |
| Task Contract + 计划 Txx/checklist 同步 | 本候选分支增加 `Implementation Tasks`、`TASK_COMPLETE`、`iw.task-progress/v1` 和 `iw task` | 已约束 task 顺序/complete/defer，但不自动同步老 Pi checklist（Core 保持 Issue Body 合同不变） | 实仓完成 T01、显式延期 T02、跨目录恢复 T02；伪造跳任务要被拒绝 |
| Settlement 逐项 decision + accepted risk | 本候选分支增加 `SETTLE` / `iw.settlement/v1` 与 Core 规则 | Host 交互/owner confirmation UI 尚未完成，逐项 inventory 与 native Pi 投影尚未全面迁移 | owner-only manual/real-resource/acceptance risk；不能标记 hard PASS；HEAD 变化失效；defer/resume 清理活跃决策 |
| defer release | 本候选分支补 remote HEAD、validation/review/contract、显式 task future settlement gate | 真实仓和复杂复核仍未验收；未完整枚举所有项目-specific gate | 不能凭 `--reason` 认定工作结束；Issue OPEN，恢复后不继承旧 risk decision |
| completion gate | 当前 HEAD validation + review + TaskProgress + typed Settlement + Native GitHub Issue close/reopen（候选分支） | 本候选分支新增显式 `iw issue close|reopen`；真实仓仍需复读确认验收、完整 acceptance inventory 与真实验收 | hard gates fail closed；Issue GitHub open/closed 单独审计 |
| portable handoff/restore | 已实现；有 fake Git 单测 | 缺双工作目录真实 GitHub 黑盒 | exact published HEAD、clean worktree、restore conflict/fail closed |
| v1 迁移、event replay/idempotency | 已实现并有测试 | 仍缺真实 GitHub failure injection | fork/duplicate/unknown outcome 不产生双事件 |
| Pi Todo/Goal/settlement UI、continuous driver | 不在 Core | **Codex / Pi 各自 Host Adapter** | host runtime 只能作为 projection；不能产生新 canonical fact |
| Codex machine adapter | `codex-workflow` 仍引用 v1 schema | **Core 验收完成后升级 Adapter** | manifest、Skill、smoke 改用 v2；端到端 Codex session 测试 |

## 本 PR 已做的有限防护

- validation 记录在检查 current HEAD 时不得因 `observed_head` 缺失而被当作有效 PASS；
- 有过 passed review 时，若 HEAD 已变化，则 completion 不再允许沿用旧 review；
- defer 前检查 worktree clean、没有 Git drift、review 无遗留问题、项目 review policy、确切 published remote HEAD；
- `wait/recheck --condition ID=waiting|resolved[:summary]` 把多个等待条件作为 event evidence 持久化，recheck 缺少任何已登记的 ID 或盲用 `--resolved` 均 fail closed；
- 实验性 Txx 任务进度 + `SETTLE` 逐项决策已入本 PR；必须由 macOS 本机 Go 测试与真实仓黑盒验证，不能据此认定完全对齐。

**不能把候选实现等于正式验收。** `iw migrate` 只处理 iw 自己的 v1 Event，**不会自动迁移 `pi-issue-state:v1` 与历史 Pi Settlement**；`.pi` 旧 Issue 导入兼容、最终 GitHub close 策略、完整 acceptance 项 inventory 仍需跟踪。验收前不能据此启动 Codex v2 正式适配。未完成项仍须由 `issue-workflow#3` 追踪并在真实仓黑盒验收后关闭。

## 正确推进顺序

1. Core：本分支已有 task/settlement/wait-item 候选 schema、CLI 和 fail-closed gates；先本机测试和修正。
2. Core：`pi-test` 真实仓黑盒；双工作目录 portable restore；迁移、review、Git drift、contract drift、operation retry。
3. Core：冻结 v2 文档、fixtures，并关闭 #3。
4. Codex：更新 manifest / SKILL / conformance smoke，验证多会话 Codex 接力；不复制 Go Core。
5. Pi-specific Goal/Todo/UI 仅供后续 `pi-workflow` 实现参考，不导入 Core。
