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

Current wire schemas remain v1 during the repository split:

- `iw.command-result/v1`
- `iw.task-contract/v1`
- `iw.workflow-event/v1`

## Current commands

```text
iw version
iw doctor [--issue N]
iw init

iw new
iw status <issue>
iw start <issue>
iw resume <issue> [--dry-run]
iw checkpoint <issue>
iw handoff <issue>
iw block <issue>
iw scope propose|accept|reject <issue>
iw review <issue>
iw fix <issue>
iw final <issue>
```

All workflow mutations support `--dry-run`. Machine callers use `--json`.

## Host adapters

Codex, Pi and future Agent integrations are separate public adapter projects. They must depend only on stable `iw` CLI/schema contracts and must not duplicate canonical workflow state logic.

The future generic adapter lifecycle is owned by `iw` itself:

```text
iw adapter list
iw adapter install <id>
iw adapter update <id>
iw adapter doctor <id>
iw adapter remove <id>
```

`iw init --adapter <id>` may provide a convenience bootstrap path. Existing `--host` remains reserved for GitHub host override and will not be repurposed for Agent selection.

Official adapters currently planned:

- `rontian/codex-workflow`
- `rontian/pi-workflow`

Adapter management is intentionally **not implemented in the bootstrap split**; the Core is completed and validated first.

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
