package domain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const workflowMarker = "<!-- iw:workflow-event:"

func ParseWorkflowEventComment(comment GitHubComment) (*WorkflowEvent, error) {
	if !strings.Contains(comment.Body, workflowMarker) {
		return nil, nil
	}
	if strings.Count(comment.Body, workflowMarker) != 1 {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow comment must contain exactly one event block")
	}
	start := strings.Index(comment.Body, workflowMarker)
	lineEndRel := strings.Index(comment.Body[start:], "\n")
	if lineEndRel < 0 {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow event header is malformed")
	}
	lineEnd := start + lineEndRel
	header := strings.TrimSpace(comment.Body[start:lineEnd])
	if header != "<!-- iw:workflow-event:v1" {
		return nil, NewWorkflowError("UNSUPPORTED_SCHEMA", fmt.Sprintf("unsupported workflow event marker %q", header))
	}
	endRel := strings.Index(comment.Body[lineEnd+1:], "-->")
	if endRel < 0 {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow event block is not closed")
	}
	end := lineEnd + 1 + endRel
	raw := strings.TrimSpace(comment.Body[lineEnd+1 : end])
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow event JSON is invalid")
	}
	required := []string{"schema", "event_id", "parent_event_id", "operation_id", "run_id", "event_type", "occurred_at", "repository", "issue", "contract_digest", "state_before", "state_after", "data"}
	for _, k := range required {
		if _, ok := keys[k]; !ok {
			return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow event missing required field: "+k)
		}
	}
	var e WorkflowEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow event JSON shape is invalid")
	}
	e.CommentID = comment.ID
	if e.Schema != WorkflowEventSchema {
		return nil, NewWorkflowError("UNSUPPORTED_SCHEMA", fmt.Sprintf("unsupported workflow event schema %q", e.Schema))
	}
	if strings.TrimSpace(e.EventID) == "" || strings.TrimSpace(e.OperationID) == "" || strings.TrimSpace(e.RunID) == "" {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "event_id, operation_id and run_id are required")
	}
	if _, err := time.Parse(time.RFC3339, e.OccurredAt); err != nil {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "occurred_at must be RFC3339")
	}
	if e.Issue <= 0 || !strings.Contains(e.Repository, "/") {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "repository/issue identity is invalid")
	}
	if !validDigest(e.ContractDigest) {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "contract_digest is invalid")
	}
	if !validState(e.StateBefore) || !validState(e.StateAfter) || !validEventType(e.EventType) {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "workflow state/event type is invalid")
	}
	if e.Data == nil {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "data must be an object")
	}
	if eventRequiresGit(e.EventType) && e.Git == nil {
		return nil, NewWorkflowError("PROTOCOL_ERROR", "git snapshot is required for "+string(e.EventType))
	}
	if e.Git != nil {
		var gitKeys map[string]json.RawMessage
		if rawGit, ok := keys["git"]; ok {
			if err := json.Unmarshal(rawGit, &gitKeys); err != nil {
				return nil, NewWorkflowError("PROTOCOL_ERROR", "git snapshot must be an object")
			}
		}
		for _, k := range []string{"branch", "head", "detached", "dirty", "changed_paths"} {
			if _, ok := gitKeys[k]; !ok {
				return nil, NewWorkflowError("PROTOCOL_ERROR", "git snapshot missing required field: "+k)
			}
		}
		if strings.TrimSpace(e.Git.Head) == "" {
			return nil, NewWorkflowError("PROTOCOL_ERROR", "git.head is required")
		}
	}
	return &e, nil
}

func validDigest(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(s, "sha256:"))
	return err == nil
}
func validState(s WorkflowState) bool {
	switch s {
	case StateReady, StateInProgress, StateBlocked, StateReview, StateFix, StateDone:
		return true
	}
	return false
}
func validEventType(t WorkflowEventType) bool {
	switch t {
	case EventStart, EventResume, EventCheckpoint, EventHandoff, EventBlocked, EventScope, EventReview, EventFix, EventFinal:
		return true
	}
	return false
}
func eventRequiresGit(t WorkflowEventType) bool {
	switch t {
	case EventStart, EventResume, EventCheckpoint, EventHandoff, EventBlocked, EventReview, EventFix, EventFinal:
		return true
	}
	return false
}
