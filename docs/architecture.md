# Architecture

## 1. Product role

`issue-workflow` is the canonical workflow engine, protocol owner, and user-facing `iw` CLI for durable AI-assisted development workflows.

The core is Agent-neutral. Codex, Pi and future coding agents integrate through separate Host Adapters that consume stable `iw` commands and machine-readable schemas.

```text
Human / CI / Agent Host Adapters
               |
               v
             iw CLI
               |
        Application Services
               |
         Workflow Domain
          /           \
       GitPort     GitHubPort
         |             |
       git CLI       gh CLI
```

## 2. Canonical truth

```text
Issue Body      = Task Contract
Issue Comments  = append-only Workflow Event Log
Git             = code truth
Go Core         = deterministic workflow rules
iw CLI          = stable human / agent interface
```

Host UI, Agent todo lists, prompts, continuous-turn state, chat history and local editor state are projections or runtime concerns. They are never canonical workflow truth.

## 3. Repository boundaries

### `rontian/issue-workflow`

Owns:

- workflow domain and state transitions;
- Task Contract and Workflow Event schemas;
- event replay, fork/conflict detection and idempotency;
- Git/GitHub fact adapters through local `git` / `gh`;
- stable CLI and machine-readable result envelopes;
- environment diagnostics;
- future generic Host Adapter lifecycle management.

Does not own:

- Codex/Pi-specific prompts, skills, extensions, UI or agent runtime behavior;
- user credentials or GitHub tokens;
- project-specific secret/config storage;
- a second GitHub HTTP/auth stack.

### Host Adapter repositories

Examples:

- `rontian/codex-workflow`
- `rontian/pi-workflow`

Adapters depend on the stable CLI/schema contract, not on Go internal packages or Core source checkout/submodules. They translate host capabilities and user intent into `iw` calls and consume structured results.

## 4. Runtime transaction

A mutating workflow command follows the deterministic transaction:

```text
read Git/GitHub facts
→ replay aggregate
→ validate preconditions
→ construct operation/event
→ locally replay candidate event
→ dry-run OR append remote event
→ verify/refresh outcome
→ return iw.command-result/v1
```

Unknown remote write outcomes are recovered using `operation_id` and operation fingerprint before retrying.

## 5. Persistence and concurrency

Workflow runtime state is reconstructed from the append-only event log. The repository does not contain `.iw` runtime databases, `current_task.md`, session state, or similar canonical state files.

GitHub comments do not provide the strong compare-and-swap primitive required for a lease. v1 therefore uses detect-and-stop semantics:

- globally unique `event_id`;
- `parent_event_id` chain;
- duplicate `operation_id` detection;
- sibling-child fork detection;
- illegal transitions and broken chains fail closed.

## 6. Local Git / GitHub boundary

`iw` invokes the user's existing local tools:

- `git` for repository identity and code facts;
- `gh` for GitHub authentication and Issue operations.

Credentials remain entirely under the user's local Git/SSH/credential-helper and GitHub CLI configuration.

`iw` must not:

- persist tokens or credentials;
- copy `gh` auth files;
- manage SSH private keys;
- commit machine-specific absolute paths or private local config;
- maintain a second GitHub credential or HTTP implementation.

`iw doctor` may only detect and report environment readiness and provide explicit local remediation instructions.

## 7. Host Adapter lifecycle

The future generic Adapter control plane belongs to `iw` itself. Canonical command direction:

```text
iw adapter list
iw adapter status <id>
iw adapter install <id>
iw adapter update <id>
iw adapter doctor <id>
iw adapter remove <id>
```

A convenience bootstrap may use:

```text
iw init --adapter codex
iw init --adapter pi
```

The existing `--host` flag is reserved for GitHub host override and must not be repurposed for Agent selection.

Adapter lifecycle is intentionally not implemented during the initial repository split. Core behavior is completed and validated first.

## 8. Compatibility rule

Current bootstrap preserves the existing v1 wire contracts:

- `iw.command-result/v1`
- `iw.task-contract/v1`
- `iw.workflow-event/v1`

Incompatible semantic or required-field changes require a new schema version. Unknown schemas fail explicitly; readers do not guess.
