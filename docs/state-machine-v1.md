# Workflow State Machine v1

> 本文描述当前兼容的 v1 状态机。后续 Agent-neutral Protocol v2 会单独设计，不在仓库拆分阶段改变这里的远端语义。

## States

```text
READY
IN_PROGRESS
BLOCKED
REVIEW
FIX
DONE
```

`DONE` 是 terminal state。

## State-changing events

```text
READY       --START-->   IN_PROGRESS
IN_PROGRESS --BLOCKED--> BLOCKED
IN_PROGRESS --REVIEW-->  REVIEW
IN_PROGRESS --FINAL-->   DONE
REVIEW      --BLOCKED--> BLOCKED
REVIEW      --FIX-->     FIX
REVIEW      --FINAL-->   DONE
FIX         --BLOCKED--> BLOCKED
FIX         --REVIEW-->  REVIEW
FIX         --FINAL-->   DONE
BLOCKED     --RESUME-->  <resume_state>
```

进入 `BLOCKED` 时 event payload 记录 `resume_state`，允许值为 `IN_PROGRESS`、`REVIEW`、`FIX`。从 BLOCKED 恢复时必须回到最近有效 BLOCKED event 指定的 resume state。

## State-neutral events

```text
CHECKPOINT
HANDOFF
SCOPE
```

`RESUME` 在 `IN_PROGRESS` / `REVIEW` / `FIX` 中也可作为 state-neutral event，用于记录新 session/Agent 的恢复。

`READY` 必须使用 START；`DONE` 不允许 RESUME。

## Command mapping

```text
iw start         -> START
iw resume        -> RESUME
iw checkpoint    -> CHECKPOINT
iw handoff       -> HANDOFF
iw block         -> BLOCKED
iw scope ...     -> SCOPE
iw review        -> REVIEW
iw fix           -> FIX
iw final         -> FINAL
```

`status`、`doctor` 不写 event。

## Blocker rules

BLOCKED 表示 workflow 无法继续，需要外部条件或明确决策。Event 至少保存 reason、next、resume_state。恢复必须显式 `resume`，不能因为出现新 commit 自动解除。

## Review / Fix

v1 中 REVIEW 是只读审查状态；finding 需要修改时进入 FIX，修复后通常回 REVIEW。是否必须经过 REVIEW 才允许 Final 属于 policy，v1 基础状态机不写死，因此 v1 允许 `IN_PROGRESS -> DONE` 和 `FIX -> DONE`。

## Scope

SCOPE 不改变 lifecycle state，但改变 aggregate gates：

- unresolved propose 阻止 Final；
- accept 更新 accepted contract digest；
- reject 关闭 proposal；
- Issue Body digest 未 reconciliation 时普通 mutation fail closed。

## Final gate

`FINAL -> DONE` 的 transition 合法只是必要条件。Final 还要求：

- event chain valid；
- 无 fork/conflict；
- contract digest 已 reconciliation；
- 无 unresolved blocker/scope proposal；
- required validation 有 pass evidence；
- Git snapshot / delivery requirements 满足；
- applicable policy 满足。

任一条件失败，不写 FINAL 成功 event。

## Replay

初始 aggregate：

```text
state = READY
last_event_id = null
```

稳定顺序 replay：schema/identity → parent chain → state_before → event payload/transition → apply → last_event_id。

断链、fork、非法 transition 或 payload 全部 fail closed。
