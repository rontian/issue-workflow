package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/result"
	"github.com/rontian/issue-workflow/internal/version"
)

func (c *CLI) render(r result.CommandResult, exit int, jsonMode bool) int {
	if jsonMode {
		enc := json.NewEncoder(c.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(r)
		return exit
	}
	switch d := r.Data.(type) {
	case version.Info:
		fmt.Fprintf(c.Stdout, "iw %s\nprotocol %s\ncommit %s\nbuild %s\n", d.Version, d.Protocol, d.Commit, d.BuildDate)
	case app.DoctorReport:
		renderDoctor(c.Stdout, d)
	case app.InitReport:
		renderDoctor(c.Stdout, d.Doctor)
		for _, a := range d.Actions {
			fmt.Fprintf(c.Stdout, "[%s] init/%s %s", a.Status, a.Kind, a.Message)
			if a.Target != "" { fmt.Fprintf(c.Stdout, " (%s)", a.Target) }
			fmt.Fprintln(c.Stdout)
		}
	case app.WorkflowReport:
		renderWorkflow(c.Stdout, d)
	case app.MutationPreview:
		fmt.Fprintf(c.Stdout, "Dry-run event %s operation=%s\n%s", d.Event.EventType, d.OperationID, d.Comment)
	case app.NewIssuePreview:
		if d.Issue != nil { fmt.Fprintf(c.Stdout, "Created Issue #%d: %s\n", d.Issue.Number, d.Issue.Title) } else { fmt.Fprintf(c.Stdout, "Dry-run Issue: %s\ncontract_id=%s\n\n%s", d.Title, d.ContractID, d.Body) }
	}
	if !r.OK && r.Error != nil { fmt.Fprintf(c.Stderr, "ERROR %s: %s\n", r.Error.Code, r.Error.Message) }
	return exit
}

func renderDoctor(w io.Writer, r app.DoctorReport) {
	for _, c := range r.Checks {
		fmt.Fprintf(w, "[%s] %-10s %-22s %s\n", c.Status, c.Category, c.ID, c.Message)
		if c.Next != "" { fmt.Fprintf(w, "       Next: %s\n", c.Next) }
	}
	fmt.Fprintf(w, "Summary: %d PASS, %d WARN, %d FAIL, %d SKIP\n", r.Summary.Pass, r.Summary.Warn, r.Summary.Fail, r.Summary.Skip)
}
