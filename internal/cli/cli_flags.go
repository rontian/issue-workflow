package cli

import (
	"flag"
	"fmt"
	"strings"
)

func flagsBeforePositionals(args []string) ([]string, error) {
	flags := []string{}
	pos := []string{}
	value := map[string]bool{"--repo": true, "--remote": true, "--host": true, "--issue": true, "--operation-id": true, "--run-id": true, "--summary": true, "--next": true, "--reason": true, "--proposal-event": true, "--title": true, "--goal": true, "--contract-id": true, "--pr": true, "--scope": true, "--out-of-scope": true, "--acceptance": true, "--constraint": true, "--validation": true, "--dependency": true, "--warning": true, "--finding": true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			flags = append(flags, a)
			name := a
			if j := strings.IndexByte(a, '='); j >= 0 { name = a[:j] }
			if value[name] && !strings.Contains(a, "=") {
				if i+1 >= len(args) { return nil, fmt.Errorf("flag %s requires a value", name) }
				i++
				flags = append(flags, args[i])
			}
		} else { pos = append(pos, a) }
	}
	return append(flags, pos...), nil
}

func registerFlags(fs *flag.FlagSet, o *options) {
	fs.StringVar(&o.Repo, "repo", o.Repo, "GitHub repository OWNER/REPO")
	fs.StringVar(&o.Remote, "remote", o.Remote, "Git remote name")
	fs.StringVar(&o.Host, "host", o.Host, "GitHub host override")
	fs.BoolVar(&o.JSON, "json", o.JSON, "machine-readable JSON output")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "preview without remote mutation")
	fs.BoolVar(&o.Verbose, "verbose", o.Verbose, "verbose diagnostics")
	fs.BoolVar(&o.Global, "global", o.Global, "global environment mode")
	fs.IntVar(&o.Issue, "issue", o.Issue, "GitHub Issue number")
	fs.BoolVar(&o.Interactive, "interactive", o.Interactive, "allow explicit interactive bootstrap")
	fs.StringVar(&o.OperationID, "operation-id", o.OperationID, "idempotency operation id")
	fs.StringVar(&o.RunID, "run-id", o.RunID, "workflow run/session id")
	fs.StringVar(&o.Summary, "summary", o.Summary, "human summary")
	fs.StringVar(&o.Next, "next", o.Next, "next action")
	fs.StringVar(&o.Reason, "reason", o.Reason, "reason")
	fs.StringVar(&o.ProposalEventID, "proposal-event", o.ProposalEventID, "scope proposal event id")
	fs.StringVar(&o.Title, "title", o.Title, "Issue title")
	fs.StringVar(&o.Goal, "goal", o.Goal, "Task Contract goal")
	fs.StringVar(&o.ContractID, "contract-id", o.ContractID, "explicit contract id")
	fs.BoolVar(&o.Pushed, "pushed", o.Pushed, "record commit as pushed in FINAL delivery")
	fs.IntVar(&o.PR, "pr", o.PR, "record pull request number in FINAL delivery")
	fs.Var(&o.ScopeItems, "scope", "repeatable Scope item")
	fs.Var(&o.OutOfScope, "out-of-scope", "repeatable Out of Scope item")
	fs.Var(&o.Acceptance, "acceptance", "repeatable Acceptance Criteria item")
	fs.Var(&o.Constraints, "constraint", "repeatable constraint")
	fs.Var(&o.Validation, "validation", "new: validation declaration; mutation: ID=pass|fail|skip[:summary]")
	fs.Var(&o.Dependencies, "dependency", "repeatable dependency/reference")
	fs.Var(&o.Warnings, "warning", "repeatable handoff warning")
	fs.Var(&o.Findings, "finding", "repeatable review finding")
}
