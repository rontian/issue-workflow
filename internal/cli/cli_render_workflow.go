package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/domain"
)

func renderWorkflow(w io.Writer, r app.WorkflowReport) {
	fmt.Fprintf(w, "Issue #%d: %s\nState: %s\nContract: %s\nContract changed: %t\nEvents: %d\n", r.Issue.Number, r.Issue.Title, r.Aggregate.State, r.Contract.Digest, r.ContractChanged, len(r.Aggregate.Events))
	if r.Aggregate.LastEvent != nil { fmt.Fprintf(w, "Latest event: %s (%s)\n", r.Aggregate.LastEvent.EventType, r.Aggregate.LastEvent.EventID) }
	fmt.Fprintf(w, "Git: %s %s dirty=%t\n", r.CurrentGit.Branch, r.CurrentGit.Head, r.CurrentGit.Dirty)
	if r.GitDrift.Changed { fmt.Fprintf(w, "Git drift: %s\n", strings.Join(r.GitDrift.Fields, ", ")) } else { fmt.Fprintln(w, "Git drift: none") }
	req := domain.ValidationRequirements(r.Contract)
	if len(req) > 0 {
		fmt.Fprintln(w, "Required validation:")
		for _, v := range req { fmt.Fprintf(w, "  %s: %s\n", v.ID, v.Text) }
	}
	if len(r.Aggregate.UnresolvedScopeProposals) > 0 { fmt.Fprintf(w, "Unresolved scope: %s\n", strings.Join(r.Aggregate.UnresolvedScopeProposals, ", ")) }
	for _, a := range r.Recovery.SuggestedActions { fmt.Fprintf(w, "Next: %s\n", a) }
}
