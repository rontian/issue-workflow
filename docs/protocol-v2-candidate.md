# iw Core v2 — Agent-neutral Issue workflow protocol (candidate)

> **Status:** implementation candidate on `fix/core-pi-parity-safety-gates`. Not a published or accepted wire-protocol release. `main` is still the stable baseline; do not update a production Codex Adapter before macOS/real GitHub acceptance.

## Truth and boundaries

- GitHub Issue **Body** is the Task Contract, not runtime state.
- GitHub Issue **Comments** are append-only `iw.workflow-event/v1|v2` history.
- Git repository HEAD, branch and worktree are code facts.
- The `iw` Go Core performs replay, legality checks, event idempotency, dry-runs and safe restores.
- Codex/Pi adapters own prompts, UI, Todo, Goal/session supervision and tool execution only. They never write raw `iw` event comments or parse human-readable error strings as their state machine.

## Machine result

All JSON commands are wrapped by `iw.command-result/v2`, including:
`schema`, `ok`, `command`, `canonical_command`, `repository`, `issue`, `lifecycle`, `state`, `warnings`, `constraints`, `next_actions`, `machine`, `data`, `error.code`.

`machine` uses `iw.machine-semantics/v1` with coded constraints, allowed actions and action IDs. `iw capabilities --json` advertises schemas and commands. Host adapters should check capabilities at startup and use `iw context <issue> --json` to recover.

## Stable task plan and Task Progress

New contracts may declare **Implementation Tasks** as `- [ ] T01: Task name` (and subsequent unique Txx identities). New tasks can be authored with repeatable `iw new ... --task "T01: description"`. The order in Issue Body is contractual. Missing Implementation Tasks preserves the existing legacy behavior, but supplies no within-Issue task progress guarantee.

```text
iw task status <issue> --json
iw task complete <issue> --task-id T01 --summary "implementation and validation evidence" [--validation v1=pass] --json
```

`iw.task-progress/v1` projects `planned`, `completed`, `current`, `next`, `all_completed` into `context` and `status`. Core replays `TASK_COMPLETE` and rejects unknown, duplicate, skipped or out-of-order IDs. Completion requires all Txx tasks complete. Defer requires unfinished Txx tasks to be explicitly settled as future scope. Host Todo is only a projection; task transitions cannot depend on the Host's private session state.

Task completion requires a clean, checkpointed HEAD. Scope acceptance resets task completion evidence against the new contract; Reopen starts a new segment. No Issue Body checklist is silently mutated.

## External wait conditions

```text
iw wait N --reason "external conditions" --next "recheck" \
  --condition ci=waiting:pending \
  --condition owner=waiting:pending

iw recheck N \
  --condition ci=resolved:build-passed \
  --condition owner=waiting:still-pending
```

For a named wait, `iw recheck` must include **every condition ID** exactly once. Omitted IDs, duplicate IDs, unknown statuses, or `--resolved` contradicting pending conditions are rejected. A partial recheck remains WAITING. All conditions resolved returns IN_PROGRESS. The legacy unstructured single-reason wait/recheck path remains for old event history.

## Settlement facts (per item)

`iw.settlement/v1` is derived exclusively from append-only `SETTLE` events; it is not a writable local state file:

```text
iw settle N --item-id task:T02 --kind implementation \
  --fact pending --disposition deferred --target defer \
  --reason "explicitly moved into the later milestone"
```

Item kinds: `implementation`, `automated_validation`, `manual_validation`, `real_resource_validation`, `review`, `git`, `contract`, `external_wait`, `acceptance`.

Fact statuses are separate from dispositions. Dispositions are `none`, `accepted_risk`, `deferred`, `kept_required`, `not_applicable`, `resolved`.

- Only **manual_validation**, **real_resource_validation**, and contract-declared `owner:` acceptance can accept risk.
- Automated test failures, confirmed findings, implementation, review, Git and contract gates cannot be risk-accepted.
- A confirmed `fail` or `findings` fact cannot be waived or deferred.
- Risk acceptance **never** changes `manual_required` to `pass`, requires exact current HEAD and a substantive owner reason (at least 20 characters).
- An owner-risk record must use explicit `--owner-confirmed` after actual owner authorization. Adapters must never infer this authorization from agent analysis. The CLI can enforce structured proof and an explicit flag, but it cannot cryptographically prove who typed a local command.
- Contract references are validated: `task:Txx`, `validation:vN`, `acceptance:aN`. Validation kind is declared by `manual:` / `real-resource:` prefixes in the Task Contract; unprefixed validations are **automated**. An acceptance criterion must explicitly begin with `owner:` to be owner-decidable.
- Deferred implementation stays incomplete. On `resume` from DEFERRED, the previous settlement projection clears; historical events remain.
- HEAD change invalidates accepted risk.
- Pending / kept-required settlement prevents `complete` or `defer`.

## Safety gates

`defer` needs clean attached worktree, current checkpointed HEAD, exact published remote branch HEAD, required validation (or allowed typed manual future settlement), no unresolved project-required review or Scope, and a complete or explicitly deferred task segment. Its result remains `DEFERRED` and GitHub Issue stays OPEN.

`complete` needs all planned implementation tasks complete, current passing validation or allowed owner-settled manual risk, no unresolved review, clean current Git state, and no unsettled completion items. Explicit `--pushed` is delivery evidence, never independent proof of a published HEAD.

`complete` may be invoked directly from legacy WAITING only with final validation/settlement evidence. Named external wait conditions must all be resolved via `recheck`. `CANCELLED` and `COMPLETED` are different; Issue close/open remains a separately audited GitHub fact, not a hidden side effect of the event. After `iw complete N` or `iw cancel N`, call `iw issue close N` explicitly; Core checks terminal state and confirms the native Issue is CLOSED. To resume an already closed Issue, call `iw reopen N` first (workflow Event) and then `iw issue reopen N` (native GitHub state). Both Issue state actions support `--dry-run`, and retries are idempotent.

`handoff --portable` requires a clean exact published HEAD and persists branch/remote/HEAD; `restore` defaults to a plan, and `restore --apply` verifies before fetching/restoring.

## Compatibility, omissions and acceptance

- `iw.task-contract/v1` remains; `Implementation Tasks` is an optional addition.
- v1 workflow events remain readable; writing v2 after v1 requires explicit `iw migrate`. No history is rewritten.
- Legacy Pi `pi-issue-state:v1` and `pi-issue-settlement:v1` comments are **not automatically converted**. This requires a separate, explicit, fail-closed import design with representative fixtures. Do not assume `iw migrate` imports Pi state.
- Owner-interactive Settlement UI and continuous execution are Host-specific, not Core.
- Current CLI still has no built-in execution of untrusted validation commands; the Agent runs real checks and submits structured evidence.
- Real macOS CI-equivalent and `rontian/pi-test` black-box results must be inspected before the candidate can be frozen. Use `scripts/verify-macos.sh`.

See [Pi workflow parity audit](pi-issue-parity-audit.md). `issue-workflow#3` remains open until all required acceptance conditions are proven.
