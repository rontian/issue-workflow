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
  iw doctor [--issue N]
  iw init

Project / recovery:
  iw project status
  iw context <issue>
  iw status <issue>

Task:
  iw new --title T --goal G --scope S --out-of-scope O --acceptance A [--validation CMD]
  iw start <issue>
  iw resume <issue> [--dry-run]
  iw checkpoint <issue> --summary S --next N [--validation v1=pass]
  iw handoff <issue> --summary S --next N [--warning W]
  iw block <issue> --reason R --next N
  iw scope propose <issue> --summary S --reason R
  iw scope accept <issue> --reason R [--proposal-event ID]
  iw scope reject <issue> --proposal-event ID --reason R
  iw review <issue> [--summary S] [--finding F]
  iw fix <issue> [--summary S]
  iw final <issue> --summary S [--validation v1=pass] [--pushed] [--pr N]

All mutating commands support --dry-run and --operation-id.
--host always means GitHub host override; Agent adapters use --adapter in the v2 adapter lifecycle.`)
}

func (c *CLI) printCommandUsage(cmd string, w io.Writer) {
	switch cmd {
	case "version":
		fmt.Fprintln(w, "Usage: iw version [--json]")
	case "capabilities":
		fmt.Fprintln(w, "Usage: iw capabilities [--json]")
	case "doctor":
		fmt.Fprintln(w, "Usage: iw doctor [--global] [--issue N] [--json]")
	case "init":
		fmt.Fprintln(w, "Usage: iw init [--dry-run] [--interactive] [--json]")
	case "project", "project status":
		fmt.Fprintln(w, "Usage: iw project status [--json]")
	case "context":
		fmt.Fprintln(w, "Usage: iw context <issue> [--json]")
	case "new":
		fmt.Fprintln(w, "Usage: iw new --title T --goal G --scope S --out-of-scope O --acceptance A [--validation CMD] [--dry-run]")
	case "scope":
		fmt.Fprintln(w, "Usage: iw scope propose|accept|reject <issue> [flags]")
	default:
		c.printUsage(w)
	}
}
