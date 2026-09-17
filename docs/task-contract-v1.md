# Task Contract v1

## 1. 定位

GitHub Issue Body 是任务契约，不是执行日志。

运行态写入 Issue Comments；代码状态写入 Git；不要把 session 状态写回仓库任务文件。

Schema ID：

```text
iw.task-contract/v1
```

## 2. Canonical Issue Body

`iw new` 创建的 Issue 使用稳定结构：

```markdown
## Goal

...

## Scope

- ...

## Out of Scope

- ...

## Constraints

- ...

## Acceptance Criteria

- [ ] ...

## Validation

- `...`

## Dependencies / References

- ...

<!-- iw:task-contract:v1
{"schema":"iw.task-contract/v1","contract_id":"..."}
-->
```

必需 section：Goal、Scope、Out of Scope、Acceptance Criteria。

可选 section：Constraints、Validation、Dependencies / References。

`iw` 只校验结构与协议字段，不用自然语言推理 Goal/Scope 是否合理；语义判断属于人类或 Agent。

## 3. Hidden contract metadata

v1 最小字段：

```json
{
  "schema": "iw.task-contract/v1",
  "contract_id": "uuid"
}
```

v1 reader 忽略未知 optional key，但不得把未知 schema 当成 v1。

## 4. Contract digest

每次 workflow writer 读取 Issue Body 后计算：

```text
contract_digest = sha256(normalize_utf8(issue_body))
```

`normalize_utf8` v1：UTF-8；CRLF 归一化为 LF；其余字符和空白保留；digest 覆盖完整 Issue Body，包括 hidden contract block。

每个 workflow event 保存 writer 当时观察到的 `contract_digest`，防止 Issue Body 在 Agent 不知情时被修改后仍继续基于旧 scope 执行。

## 5. Contract change

Issue 已有 workflow event 后，当前 Body digest 与 latest accepted digest 不同时：

- `status` / `doctor` 等只读命令继续，但报告 `CONTRACT_CHANGED`；
- 普通 mutating command fail closed；
- 新 Body 不会自动成为授权 scope。

显式 reconciliation 通过 `scope` 协议完成。

## 6. Scope reconciliation

`scope propose` 记录待决范围变化，不修改 lifecycle state；未解决 proposal 阻止 Final。

`scope accept` 显式接受当前 Issue Body 作为新 contract，并记录 previous/new contract digest 与 reason。

`scope reject` 关闭 proposal，但不会接受外部修改过的 Body；若 Body digest 仍变化，仍保持 `CONTRACT_CHANGED`。

## 7. Checkbox policy

Acceptance Criteria checkbox 属于 Task Contract 内容。Workflow v1 不自动勾选 Issue Body checkbox；完成事实写入 FINAL event。人类手动改 checkbox 等同普通 Contract Change，需要显式 reconciliation。

## 8. Validation declaration

`Validation` section 声明任务希望执行的验证。v1 不自动执行 Issue 中任意文本命令。

Agent/人类负责执行验证；`iw` 保存结构化 validation result，并在 Final gate 检查 required validation 是否有 pass 记录。
