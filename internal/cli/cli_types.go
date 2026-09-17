package cli

import (
	"io"
	"strings"

	"github.com/rontian/issue-workflow/internal/app"
)

type CLI struct {
	Doctor     *app.DoctorService
	Init       *app.InitService
	Workflow   *app.WorkflowService
	Project    *app.ProjectService
	Context    *app.ContextService
	Stdout     io.Writer
	Stderr     io.Writer
	Getwd      func() (string, error)
	IsTerminal func() bool
}

type stringList []string
func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

type options struct {
	Repo, Remote, Host                                                                    string
	JSON, DryRun, Verbose, Global                                                         bool
	Issue                                                                                 int
	Interactive                                                                           bool
	OperationID, RunID, Summary, Next, Reason, ProposalEventID                            string
	Title, Goal, ContractID                                                               string
	Pushed                                                                                bool
	PR                                                                                    int
	ScopeItems, OutOfScope, Acceptance, Constraints, Validation, Dependencies, Warnings, Findings stringList
	ScopeAction                                                                           string
}
