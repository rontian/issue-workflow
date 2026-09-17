# Workflow Metadata v1

## 1. 目标

GitHub Issue comment 同时服务人类和程序：人类阅读 Markdown marker/summary，程序只读取 hidden JSON event。

Schema：

```text
iw.workflow-event/v1
```

## 2. Comment envelope

```markdown
[CHECKPOINT]

完成 xxx，验证 yyy 通过。

<!-- iw:workflow-event:v1
{"schema":"iw.workflow-event/v1", ...}
-->
```

Parser 规则：

- 普通 comment 完全忽略；
- 声明 v1 但 JSON 非法、缺字段或存在多个 event block => `PROTOCOL_ERROR`；
- 未知 `iw:workflow-event:vN` 不得静默按 v1 解释；
- Human marker 不是状态来源。

## 3. Canonical event

```json
{
  "schema": "iw.workflow-event/v1",
  "event_id": "uuid",
  "parent_event_id": "uuid-or-null",
  "operation_id": "uuid",
  "run_id": "uuid",
  "event_type": "CHECKPOINT",
  "occurred_at": "2026-09-15T04:00:00Z",
  "repository": "owner/repo",
  "issue": 123,
  "contract_digest": "sha256:...",
  "state_before": "IN_PROGRESS",
  "state_after": "IN_PROGRESS",
  "git": {
    "branch": "main",
    "head": "abc123...",
    "detached": false,
    "dirty": true,
    "changed_paths": ["internal/domain/state.go"]
  },
  "data": {
    "summary": "..."
  }
}
```

所有 event required：`schema`、`event_id`、`parent_event_id`、`operation_id`、`run_id`、`event_type`、`occurred_at`、`repository`、`issue`、`contract_digest`、`state_before`、`state_after`、`data`。`git` 是否 required 由 event type 决定。

## 4. Identity / idempotency

`event_id` 全局唯一。

`parent_event_id` 指向 writer 观察到的 latest valid event；第一条 event 为 null。它用于断链和 fork 检测。

`operation_id` 是一次逻辑 mutation 的幂等键。未知写入结果时，retry 必须先扫描远端 events：相同 operation 且 fingerprint 兼容 => 已成功；相同 ID 但内容冲突 => `CONFLICT`；不存在才允许重试 append。

`run_id` 表示一次 Agent/人工执行会话，用于 provenance/handoff，不是强 lease。

## 5. Contract digest

普通 mutation 要求当前 Issue Body digest 等于 latest accepted contract digest。否则 `CONTRACT_CHANGED`，必须显式 scope reconciliation。

## 6. Git snapshot

Event 中的 `git` 是事件发生时的历史快照，不是当前事实。字段包括 branch、head、detached、dirty、changed_paths。

START、RESUME、CHECKPOINT、HANDOFF、BLOCKED、REVIEW、FIX、FINAL 应记录 Git snapshot。恢复时必须重新读取本地 Git 当前事实并比较 drift。

## 7. Event-specific data

CHECKPOINT：summary、next、validation。

HANDOFF：summary、next、warnings。

BLOCKED：reason、next、resume_state。

SCOPE：action (`propose|accept|reject`)、按 action 所需的 summary/reason/digest/proposal fields。

FINAL：summary、validation、delivery（commit/pushed/pr）。

`data` 可增加 optional key；reader 忽略未知 optional key，但校验当前 event type 的 required key。

## 8. Ordering / replay

Issue comments 必须完整分页读取并按稳定创建顺序 replay：

1. repository/issue identity 匹配；
2. event_id 唯一；
3. operation_id 不冲突；
4. parent_event_id 等于 last_event_id；
5. state_before 等于 current state；
6. transition/payload 合法；
7. apply；
8. 更新 last_event_id。

两个 event 指向同一 parent => `EVENT_FORK`。不得因为 GitHub comment 有先后顺序就自动选择某一分支；mutations fail closed。

## 9. Compatibility

- v1 writer 只写 `iw.workflow-event/v1`；
- 已发布字段语义不可改变；
- 可新增 optional key；
- required 字段或不兼容语义变化必须新 schema；
- unknown schema 必须显式失败。
