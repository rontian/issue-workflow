package cli

import (
	"fmt"
	"io"
)

func (c *CLI) printUsage(w io.Writer) {
	fmt.Fprintln(w, `iw - Agent-neutral GitHub Issue workflow CLI

Environment / protocol:
  iw version
  iw capabilities
  iw doctor [--global] [--issue N] [--adapter ID]
  iw init [--interactive] [--dry-run] [--adapter ID]

Project / task read model:
  iw project status
  iw tasks
  iw next
  iw context <issue>
  iw status <issue>

Task creation / migration:
  iw new --title T --goal G --scope S --out-of-scope O --acceptance A [--validation CMD]
  iw migrate <issue>

Lifecycle:
  iw start <issue>
  iw resume <issue>
  iw pause <issue> --reason R
  iw wait <issue> --reason R --next N
  iw recheck <issue> [--resolved] [--reason R]
  iw defer <issue> --reason R --next N
  iw checkpoint <issue> --summary S --next N [--validation ID=status[:summary]]
  iw handoff <issue> --summary S --next N [--warning W] [--portable]
  iw restore <issue> [--apply]
  iw block <issue> --reason R --next N
  iw scope propose <issue> --summary S --reason R
  iw scope accept <issue> --reason R [--proposal-event ID]
  iw scope reject <issue> --proposal-event ID --reason R
  iw review <issue> [--summary S] [--finding F]
  iw fix <issue> --summary S
  iw complete <issue> --summary S [--validation ID=status[:summary]] [--pushed] [--pr N]
  iw final <issue> --summary S ...        # compatibility alias for complete
  iw cancel <issue> --reason R
  iw reopen <issue> --reason R

Adapter lifecycle:
  iw adapter list
  iw adapter status <id>
  iw adapter doctor <id>
  iw adapter install <id> [--dry-run]
  iw adapter update <id> [--dry-run]
  iw adapter remove <id> [--dry-run]

Workflow mutations and iw new support --dry-run and --operation-id.
Portable handoff verifies that the exact HEAD is published; restore is plan-only unless --apply is given.
Adapter install/update/remove support --dry-run and never make Host Adapter state canonical workflow truth.
--host always means GitHub host override; Agent adapters use --adapter.`)
}

func (c *CLI) printCommandUsage(cmd string, w io.Writer) {
	switch cmd {
	case "version":
		fmt.Fprintln(w, "Usage: iw version [--json]")
	case "capabilities":
		fmt.Fprintln(w, "Usage: iw capabilities [--json]")
	case "doctor":
		fmt.Fprintln(w, "Usage: iw doctor [--global] [--issue N] [--adapter ID] [--json]")
	case "init":
		fmt.Fprintln(w, "Usage: iw init [--dry-run] [--interactive] [--adapter ID] [--json]")
	case "project", "project status":
		fmt.Fprintln(w, "Usage: iw project status [--json]")
	case "tasks":
		fmt.Fprintln(w, "Usage: iw tasks [--json]")
	case "next":
		fmt.Fprintln(w, "Usage: iw next [--json]")
	case "context":
		fmt.Fprintln(w, "Usage: iw context <issue> [--json]")
	case "status":
		fmt.Fprintln(w, "Usage: iw status <issue> [--json]")
	case "new":
		fmt.Fprintln(w, "Usage: iw new --title T --goal G --scope S --out-of-scope O --acceptance A [--validation CMD] [--dry-run] [--operation-id ID]")
	case "migrate":
		fmt.Fprintln(w, "Usage: iw migrate <issue> [--dry-run] [--operation-id ID]")
	case "restore":
		fmt.Fprintln(w, "Usage: iw restore <issue> [--apply] [--json]")
	case "handoff":
		fmt.Fprintln(w, "Usage: iw handoff <issue> --summary S --next N [--warning W] [--portable] [--dry-run] [--operation-id ID]")
	case "scope":
		fmt.Fprintln(w, "Usage: iw scope propose|accept|reject <issue> [flags]")
	case "adapter":
		fmt.Fprintln(w, "Usage: iw adapter list | status|doctor|install|update|remove <id> [--dry-run] [--json]")
	case "start", "resume", "pause", "wait", "recheck", "defer", "checkpoint", "block", "review", "fix", "complete", "final", "cancel", "reopen":
		fmt.Fprintf(w, "Usage: iw %s <issue> [flags]\n", cmd)
	default:
		c.printUsage(w)
	}
}
