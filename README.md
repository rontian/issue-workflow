# issue-workflow

`issue-workflow` is the canonical Agent-neutral development workflow engine and the `iw` CLI.

The product boundary is intentionally simple:

```text
User / CI / Agent adapters
          |
          v
         iw
   Go Core + Protocol
      |          |
     git         gh
      |          |
   local Git   GitHub Issues
```

## Install

```bash
go install github.com/rontian/issue-workflow/cmd/iw@latest
```

Runtime dependencies:

- Go is only required for `go install`; the installed `iw` binary is standalone.
- `git` must already be configured by the user.
- `gh` must already be installed and authenticated by the user.

`iw` does **not** store GitHub tokens, copy credentials, manage SSH keys, or maintain a second GitHub HTTP client. It invokes the user's local `git` / `gh` configuration and `iw doctor` reports missing or invalid local environment state.

## Canonical facts

```text
Issue Body      = Task Contract
Issue Comments  = append-only Workflow Event Log
Git             = code truth
Go Core         = deterministic workflow rules
iw CLI          = stable human / agent interface
```

Current v2 schema baseline (new task/settlement additions below are **candidate branch only**, not on main):

- `iw.command-result/v2` with `iw.machine-semantics/v1`
- `iw.context/v1` and `iw.capabilities/v1`
- `iw.task-contract/v1` (read compatibility)
- `iw.workflow-event/v2` (v1 event reader + explicit `iw migrate`)
- `iw.adapter/v1` for official Host Adapter manifests

## CLI commands (implemented, not yet fully black-box accepted)

```text
iw version --json
iw capabilities --json
iw doctor [--global] [--issue N] [--adapter codex] --json
iw init [--interactive] [--dry-run] [--adapter codex]
iw project status --json
iw tasks --json
iw next --json
iw context <issue> --json
iw status <issue> --json

iw new --title T --goal G --scope S --out-of-scope O --acceptance A [--task "T01: name"]
iw migrate <issue>
iw start|resume|pause|wait|recheck|defer <issue>
iw checkpoint|handoff|restore <issue>
iw block|review|fix|complete|cancel|reopen <issue>
iw scope propose|accept|reject <issue>
iw task status <issue> | iw task complete <issue> --task-id T01 --summary S
iw settle <issue> --item-id task:T02 --kind implementation --fact pending --disposition deferred --target defer --reason R

iw adapter list
iw adapter status|doctor|install|update|remove codex
```

Use `iw <command> --help` for the complete flag contract. Mutations support `--dry-run` and `--operation-id`. Restore is plan-only unless `--apply` is given. Portable handoff requires a clean, published exact HEAD.

## Host adapters

The Core owns adapter registry/manifest validation and managed installation. It does not contain Codex/Pi-specific prompt, Todo or Goal implementations.

```text
iw adapter list
iw adapter doctor codex
iw adapter install codex --dry-run
iw adapter install codex
```

**Current compatibility warning:** `rontian/codex-workflow` on its older `main` still requires `iw.command-result/v1` and `iw.workflow-event/v1`. The Core correctly rejects that manifest with `ADAPTER_INCOMPATIBLE`; do not work around the gate or install an incompatible Skill manually.

**Delivery status:** the original Core v2 main had a green CI on 2026-09-17; this **candidate branch has not been verified on macOS** because hosted GitHub Actions are unavailable. Core #3 remains open. Baseline but full real-GitHub black-box acceptance, fine-grained Txx task transition, per-item Settlement, and multi-item waiting semantics still need work. Core #3 remains the acceptance contract. See [Pi workflow parity audit](docs/pi-issue-parity-audit.md) and [candidate v2 protocol](docs/protocol-v2-candidate.md).

## macOS local verification (no GitHub Actions)

From a local clone of this PR branch:

```bash
git fetch origin fix/core-pi-parity-safety-gates
git switch fix/core-pi-parity-safety-gates
bash scripts/verify-macos.sh
```

This runs tests/race/vet/build/install and protocol smoke locally; **no GitHub writes**. Reports go to `~/Library/Logs/iw-verification/<UTC-stamp>-<pid>/` and are kept for inspection. After local PASS, opt into the isolated real GitHub test:

```bash
IW_TEST_CONFIRM=YES bash scripts/verify-macos.sh --live
```

This uses only newly created `[iw-blackbox]` Issues and a new test branch in `rontian/pi-test` and leaves them for investigation by default; **never runs on `issue-workflow` or `.pi`**. Append `--cleanup` to close/delete only artifacts created by that invocation on full success. Review report logs before sharing; do not post credentials, private paths or private issue contents. See also `scripts/verify-macos.sh --help`.


## Public / local boundary

This repository and official adapter repositories are designed to remain safe as public source repositories.

Do not design repository-owned storage for user secrets, tokens, credentials, machine-specific absolute paths, private repository configuration, or local runtime caches. Local credentials and Git/GitHub identity remain under the user's existing `git` / `gh` configuration.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/iw
```

Repository split work is tracked in Issue #1.

### Multiple named external waiting conditions

Use `iw wait <issue> --reason 'upstream dependency' --next 'recheck' --condition ci=waiting:pending --condition owner=waiting:pending`. The next `iw recheck <issue> --condition ci=resolved:build-passed --condition owner=waiting:still-pending --json` must report **every** condition. The Core keeps `WAITING` while any remain unresolved; a blind `--resolved` is rejected for named waits. Legacy single-reason waits retain their old CLI behavior.
