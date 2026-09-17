# CLI Contract v1

## Binary

```text
iw
```

Human-readable output is the default. `--json` returns exactly one JSON document using:

```text
iw.command-result/v1
```

## Command result envelope

```json
{
  "schema": "iw.command-result/v1",
  "ok": true,
  "command": "status",
  "repository": "owner/repo",
  "issue": 123,
  "state": "IN_PROGRESS",
  "warnings": [],
  "constraints": [],
  "next_actions": [],
  "data": {},
  "error": null
}
```

Machine consumers must use structured fields and error codes rather than parsing human output.

## Exit codes

```text
0 success
2 invalid invocation
3 environment/dependency
4 GitHub/auth/availability
5 workflow transition/gate
6 validation
7 conflict/stale state
8 protocol/schema
9 internal
```

Specific machine meaning is carried by `error.code`.

## Current environment commands

```text
iw version
iw doctor [--global] [--issue N]
iw init [--interactive] [--dry-run]
```

`--host` means GitHub host override. This meaning is compatibility-sensitive and must not be reused for Host Adapter selection.

Future Agent Adapter bootstrap will use a separate `--adapter` option / `iw adapter` command family.

## Current workflow commands

```text
iw new
iw status <issue>
iw start <issue>
iw resume <issue>
iw checkpoint <issue>
iw handoff <issue>
iw block <issue>
iw scope propose|accept|reject <issue>
iw review <issue>
iw fix <issue>
iw final <issue>
```

All remote workflow mutations support `--dry-run`. Logical mutations may receive explicit `--operation-id` for safe retry/recovery.

## Security/runtime boundary

`iw` executes local `git` and `gh` directly with argv; it does not build shell command strings for workflow data. Issue bodies/comments are passed to `gh` through stdin where applicable.

Authentication is owned by the user's local `gh` configuration. `iw` never returns, persists or republishes credentials.

## Compatibility

v1 machine callers may rely on the envelope schema and published error-code meanings. Incompatible changes require a new result schema. New optional data fields may be added without changing v1 semantics.
