package app

import "github.com/rontian/issue-workflow/internal/domain"

type MutationOptions struct {
	WorkflowOptions
	DryRun          bool
	OperationID     string
	RunID           string
	Summary         string
	Next            string
	Reason          string
	Warnings        []string
	Validation      []domain.ValidationRecord
	Findings        []string
	ScopeAction     string
	ProposalEventID string
	Pushed          bool
	PR              int
}

type MutationPreview struct {
	Event       domain.WorkflowEvent `json:"event"`
	Comment     string               `json:"comment"`
	OperationID string               `json:"operation_id"`
	DryRun      bool                 `json:"dry_run"`
}

type NewIssueOptions struct {
	WorkflowOptions
	DryRun       bool
	Title        string
	Goal         string
	Scope        []string
	OutOfScope   []string
	Constraints  []string
	Acceptance   []string
	Validation   []string
	Dependencies []string
	ContractID   string
}

type NewIssuePreview struct {
	Title      string                     `json:"title"`
	Body       string                     `json:"body"`
	ContractID string                     `json:"contract_id"`
	DryRun     bool                       `json:"dry_run"`
	Issue      *domain.GitHubIssueDetails `json:"issue,omitempty"`
}
