package domain

import (
	"fmt"
	"sort"
)

const (
	TaskContractSchema  = "iw.task-contract/v1"
	WorkflowEventSchema = "iw.workflow-event/v1"
)

type WorkflowState string

const (
	StateReady      WorkflowState = "READY"
	StateInProgress WorkflowState = "IN_PROGRESS"
	StateBlocked    WorkflowState = "BLOCKED"
	StateReview     WorkflowState = "REVIEW"
	StateFix        WorkflowState = "FIX"
	StateDone       WorkflowState = "DONE"
)

type WorkflowEventType string

const (
	EventStart      WorkflowEventType = "START"
	EventResume     WorkflowEventType = "RESUME"
	EventCheckpoint WorkflowEventType = "CHECKPOINT"
	EventHandoff    WorkflowEventType = "HANDOFF"
	EventBlocked    WorkflowEventType = "BLOCKED"
	EventScope      WorkflowEventType = "SCOPE"
	EventReview     WorkflowEventType = "REVIEW"
	EventFix        WorkflowEventType = "FIX"
	EventFinal      WorkflowEventType = "FINAL"
)

type WorkflowError struct{ Code, Message string }

func (e *WorkflowError) Error() string        { return e.Message }
func NewWorkflowError(code, msg string) error { return &WorkflowError{Code: code, Message: msg} }
func WorkflowErrorCode(err error) string {
	if e, ok := err.(*WorkflowError); ok {
		return e.Code
	}
	return ""
}

type TaskContract struct {
	Schema                 string `json:"schema"`
	ContractID             string `json:"contract_id"`
	Digest                 string `json:"digest"`
	Goal                   string `json:"goal"`
	Scope                  string `json:"scope"`
	OutOfScope             string `json:"out_of_scope"`
	Constraints            string `json:"constraints,omitempty"`
	AcceptanceCriteria     string `json:"acceptance_criteria"`
	Validation             string `json:"validation,omitempty"`
	DependenciesReferences string `json:"dependencies_references,omitempty"`
}

type EventGitSnapshot struct {
	Branch       *string  `json:"branch"`
	Head         string   `json:"head"`
	Detached     bool     `json:"detached"`
	Dirty        bool     `json:"dirty"`
	ChangedPaths []string `json:"changed_paths"`
}

type WorkflowEvent struct {
	Schema         string            `json:"schema"`
	EventID        string            `json:"event_id"`
	ParentEventID  *string           `json:"parent_event_id"`
	OperationID    string            `json:"operation_id"`
	RunID          string            `json:"run_id"`
	EventType      WorkflowEventType `json:"event_type"`
	OccurredAt     string            `json:"occurred_at"`
	Repository     string            `json:"repository"`
	Issue          int               `json:"issue"`
	ContractDigest string            `json:"contract_digest"`
	StateBefore    WorkflowState     `json:"state_before"`
	StateAfter     WorkflowState     `json:"state_after"`
	Git            *EventGitSnapshot `json:"git,omitempty"`
	Data           map[string]any    `json:"data"`
	CommentID      int64             `json:"-"`
}

type GitHubIssueDetails struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url,omitempty"`
}

type GitHubComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
}

type WorkflowAggregate struct {
	State                    WorkflowState     `json:"state"`
	Events                   []WorkflowEvent   `json:"events"`
	LastEvent                *WorkflowEvent    `json:"last_event,omitempty"`
	LatestGit                *EventGitSnapshot `json:"latest_git,omitempty"`
	AcceptedContractDigest   string            `json:"accepted_contract_digest,omitempty"`
	UnresolvedScopeProposals []string          `json:"unresolved_scope_proposals"`
	BlockReason              string            `json:"block_reason,omitempty"`
	ResumeState              WorkflowState     `json:"resume_state,omitempty"`
}

type GitDrift struct {
	Changed   bool              `json:"changed"`
	Fields    []string          `json:"fields"`
	Persisted *EventGitSnapshot `json:"persisted,omitempty"`
	Current   GitSnapshot       `json:"current"`
}

func CompareGitSnapshot(p *EventGitSnapshot, c GitSnapshot) GitDrift {
	d := GitDrift{Persisted: p, Current: c, Fields: []string{}}
	if p == nil {
		return d
	}
	branch := ""
	if p.Branch != nil {
		branch = *p.Branch
	}
	if branch != c.Branch {
		d.Fields = append(d.Fields, "branch")
	}
	if p.Head != c.Head {
		d.Fields = append(d.Fields, "head")
	}
	if p.Detached != c.Detached {
		d.Fields = append(d.Fields, "detached")
	}
	if p.Dirty != c.Dirty {
		d.Fields = append(d.Fields, "dirty")
	}
	a := append([]string(nil), p.ChangedPaths...)
	b := append([]string(nil), c.ChangedPaths...)
	sort.Strings(a)
	sort.Strings(b)
	if fmt.Sprint(a) != fmt.Sprint(b) {
		d.Fields = append(d.Fields, "changed_paths")
	}
	d.Changed = len(d.Fields) > 0
	return d
}
