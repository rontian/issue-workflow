# Core v2 — Issue #3 Acceptance / Evidence / Gap Matrix

> Baseline audited 2026-10-29: [Task Contract #3](https://github.com/rontian/issue-workflow/issues/3), [handoff](https://github.com/rontian/issue-workflow/issues/3#issuecomment-6489611472), [Draft PR #5](https://github.com/rontian/issue-workflow/pull/5). Baseline `main=40f98c1`; acceptance candidate was `e8d0b80`. This matrix tracks the **candidate branch**, not a release. After every implementation change, collect fresh evidence against the exact new HEAD. Do not mark Issue #3 completed from the older PASS alone.

Status vocabulary:
- **Candidate / unit**: relevant implementation and tests exist; not full acceptance.
- **Previous live PASS**: user-supplied macOS real-GitHub evidence on `e8d0b80` only.
- **Pending**: no sufficient evidence for the full acceptance criterion.
- **Release-only**: intentionally gated on authorization and merge.

| #3 Acceptance Criterion | Core implementation / schema | Existing test or fixture | Real macOS / pi-test evidence | Gap / next gate | State |
| --- | --- | --- | --- | --- | --- |
| Lifecycle and v2 schema frozen | `internal/domain/{lifecycle_v2,workflow,metadata,replay}.go`, `docs/protocol-v2-candidate.md` | `internal/domain/{replay,metadata}_test.go` | v2 positive transitions on test Issue #5 | Protocol fixture inventory, unknown schema/fork negative live | Candidate / unit |
| Machine result understood without prose | `internal/result/result.go`, `internal/cli/cli_render_*.go` | `internal/result/result_test.go`, new migration/contract-drift/action tests | version/capability JSON smoke on previous HEAD | Check `machine.allowed_actions`, constraints and coded next actions for all commands against new HEAD | Candidate / unit |
| capabilities/context/project/tasks/next | `internal/app/{capabilities,context,project,tasks}.go` | `internal/app/{project_contract,tasks,workflow}_test.go` | context from second checkout on test Issue #5 | Field-by-field recovery/schema fixture & environment readiness; strict next action | Candidate / unit |
| Pause / wait / recheck / defer / block / complete / cancel / reopen | `internal/app/mutation_*.go`, `internal/domain/{replay,wait_conditions,settlement}.go` | state, wait, defer, settlement, gate tests | selected wait/defer/complete/reopen/cancel paths on test Issue #5 | Fail-closed transition matrix and no-event assertions for rejected writes | Previous live PASS + pending negatives |
| v1 reads and explicit safe migration | `internal/domain/replay.go`, `internal/app/mutation_plan.go` | `TestReplayRequiresExplicitV1ToV2Migration`, `TestReplayV2MigrationLifecycle` | no full legacy live migration receipt | Legacy v1 fixtures, unknown v1 state and migration anchor replay/no-fork | Pending |
| Validation/review evidence and completion gate | `internal/domain/validation.go`, `internal/app/mutation_plan.go`, `internal/app/settlement_gate.go` | `scope_validation_test.go`, `validation_head_test.go`, gate tests | positive review/complete on test Issue #5 | Stale HEAD, stale review, unresolved findings, current evidence negatives in new isolated Issue | Candidate / unit |
| Portable handoff/restore site recovery | `internal/app/restore.go`, `internal/adapters/gitcli/portable.go` | `portable_test.go`, `portable_shallow_integration_test.go` | two shallow checkouts on test Issue #5 | Dirty apply, wrong remote/HEAD, remote change and no-reset live; verify required context fields | Previous live PASS + pending negatives |
| Project/environment contract and privacy | `internal/app/{project,doctor,doctor_project}.go`, `internal/domain/environment.go` | `project_contract_test.go`, `doctor_test.go` | doctor on test Issue #5 | Malicious config/credential/absolute path cases; docs audit | Candidate / unit |
| Adapter manifest/registry/minimum lifecycle | `internal/app/adapter.go` | `adapter_test.go`, `internal/cli/adapter_cli_test.go` | no current full adapter install/unmanaged test receipt | Local official registry, manifest compatibility, dry-run, unmanaged target, remove/update rollback | Pending |
| All mutations dry-run and operation-idempotency | `internal/app/mutation_commands.go`, `internal/app/mutation_helpers_*.go` | `mutation_idempotency_test.go`, `mutation_core_test.go` | start replay and retry of transient write/readback errors on test Issue #5 | All mutation commands (including create/native actions); conflict/retry/unknown-outcome table and no-event assertions | Candidate / unit |
| unit / race / vet / build / install smoke | `scripts/verify-macos.sh` | Go suite | Previous user macOS arm64 Go 1.26.6 `Fail count: 0` at `e8d0b80` | Re-run on current HEAD | Previous live PASS (old HEAD) |
| go install @latest after main | `go.mod`, `cmd/iw` | not applicable | not yet authorized | Merge and tag/release strategy required; `@latest` is not equal to `@main` | Release-only |
| pi-test real GitHub black box | `scripts/verify-macos.sh`, `scripts/verify-negative-macos.sh` | local synthetic/integration tests | Issue #5: CLOSED, 15 v2 Events, no duplicated operation ID, old HEAD | Run **new** isolated negative script with explicit consent, confirm no-event checks, preserve #5 audit | Previous positive PASS, new negatives pending |
| README/architecture/CLI/protocol/migration/testing docs | `README.md`, `docs/*.md` | manual review | no full doc conformance receipt | Correct stale evidence, publish schema/error catalog, finish recovery/migration runbook | Pending |

## Negative acceptance matrix (new script)

`scripts/verify-negative-macos.sh` creates **new** `[iw-negative]` Issues and `iw-negative-*` test branch only in `rontian/pi-test`; it does not touch test Issue #5. Default invocation is local only. Live requires **both** `IW_TEST_CONFIRM=YES` and `--live`. Expected errors and event counts are written to a results CSV. The second, sacrificial Issue alone receives an intentionally unknown event marker.

| Live check | Expected machine error | Additional invariant |
| --- | --- | --- |
| same operation-id, different payload | `CONFLICT` | No new Workflow Event |
| mutation after Contract Body digest drift | `CONTRACT_CHANGED` | No event until explicit scope acceptance |
| dry-run mutation | success | No new Event |
| portable handoff on dirty worktree | `STALE_GIT_STATE` | No new Event |
| complete after local HEAD changes with prior passed Review | `REVIEW_REQUIRED` (Review HEAD gate precedes Git drift) | No new Event; independent Git drift is covered by `TestGitHeadDriftWithoutExistingReviewFailsClosed` |
| portable handoff with unpublished HEAD | `STALE_GIT_STATE` | No new Event |
| complete with stale review HEAD | `REVIEW_REQUIRED` | No new Event |
| complete with stale validation HEAD | `VALIDATION_FAILED` | No new Event |
| restore with moved remote branch HEAD | `RESTORE_CONFLICT` | No new Event; no local reset; checkpoint new HEAD before explicit cancel/native close |
| unsupported event marker on sacrificial Issue | `UNSUPPORTED_SCHEMA` | Read and write fail closed; no additional Event |

**Not yet covered deterministically by this live script:** unknown/ambiguous write delivery under injected transport faults, malformed *valid-marker* event chain/fork, comprehensive v1 migration, concurrent mutations, and exhaustive adapter manager/security conformance. The fake-GitHub Go suite covers selected unknown-write outcomes; do **not** claim full negative acceptance on the basis of the script alone.

## 2026-10-29 macOS retest checkpoint (failed, no live writes)

At branch `0473d534`, user ran local `verify-macos.sh` (`Fail count: 2`) and local-only portions of the negative verifier twice (each `Fail count: 1`). The sole failing test was `TestStaleReviewAndValidationHeadsDoNotAppendCompletion`, which incorrectly expected `STALE_GIT_STATE/7` when a previously passed review belonged to an old HEAD. The Core correctly returned `REVIEW_REQUIRED/6` before the general Git drift check. Unit/race stopped the opt-in live negative run before any new Issue/branch was created.

Fixed by test commits `6c02af6` (priority assertion + independent no-review Git-drift test) and `361df42` (live script expectation and later checkpoint before native close). These fixes have **not yet passed the user's macOS retest**; no completion checklist may be checked from this record.

## 2026-10-29 successful local retest and partial pi-test negative run

User's subsequent Go 1.26.6/darwin arm64 results: two targeted HEAD-gate tests **PASS**; `verify-macos.sh` local shell/compile/unit/race/vet/build/install/schema/help all **PASS**, `Fail count: 0`; `verify-negative-macos.sh` local shell/unit/build **PASS**, `Fail count: 0`.

Explicitly authorized `IW_TEST_CONFIRM=YES ... --live` created dedicated `rontian/pi-test#6` and `iw-negative-20261029T083558Z-17022`. Recorded five canonical events (START, CHECKPOINT, REVIEW, SCOPE, REVIEW). Operation-ID conflict, dry-run no-append, Contract drift/reconcile, dirty portable handoff all **PASS** including invariant checks (no event added on rejection). At `review-precedes-git-drift`, machine result was `GITHUB_UNAVAILABLE/4` (expected `REVIEW_REQUIRED`); the external Event count remained **5 -> 5**. This is **an incomplete infrastructure-dependent check, not a completion-gate PASS**. Issue #6 and branch remain as isolated audit evidence; the test did not reach poison-schema, restore, or final close.

Following that evidence, branch commits `29dc5f5` and `f201fb2` add one bounded retry of failed **read-only** paginated Issue comments; `d10c9be` tests read retry, permanent failure, malformed page, cancellation and no retry of comment writes; `d061004` classifies `GITHUB_UNAVAILABLE` in the live harness as `INFRA_FAIL` with minimal diagnostics. **This newer HEAD has not yet been retested on macOS**, and a second live run will create a new, separately isolated test Issue/branch rather than silently resuming #6.

## Next verification

```bash
git fetch origin fix/core-pi-parity-safety-gates
git switch fix/core-pi-parity-safety-gates
bash scripts/verify-macos.sh
bash scripts/verify-negative-macos.sh

# Separate opt-in only after local PASS, dedicated pi-test only:
IW_TEST_CONFIRM=YES bash scripts/verify-negative-macos.sh --live

# On a future new run, --cleanup removes only the new test branch on success;
# the closed test Issue comments intentionally remain auditable.
```

Collect `results.csv`, failing-step JSON and a fresh GitHub event-count audit **without** credentials/private paths. The user-run report must identify the exact branch HEAD. Neither the acceptance checklist in #3 nor PR status may be upgraded before evidence is attached.
