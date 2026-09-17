package domain

import (
	"fmt"
	"strings"
	"testing"
)

func digest(ch byte) string { return "sha256:" + strings.Repeat(string(ch), 64) }
func eventJSON(id, parent, op, typ, before, after string, data string) string {
	p := "null"
	if parent != "" { p = fmt.Sprintf("%q", parent) }
	return fmt.Sprintf(`{"schema":"iw.workflow-event/v1","event_id":%q,"parent_event_id":%s,"operation_id":%q,"run_id":"r1","event_type":%q,"occurred_at":"2026-09-15T04:00:00Z","repository":"o/r","issue":7,"contract_digest":%q,"state_before":%q,"state_after":%q,"git":{"branch":"main","head":"abcdef","detached":false,"dirty":false,"changed_paths":[]},"data":%s}`, id, p, op, typ, digest('a'), before, after, data)
}
func comment(body string) GitHubComment { return GitHubComment{ID: 1, Body: body, CreatedAt: "2026-09-15T04:00:00Z"} }
func wrap(raw string) string { return "[X]\n\n<!-- iw:workflow-event:v1\n" + raw + "\n-->" }
func TestParseWorkflowEventComment(t *testing.T) {
	e, err := ParseWorkflowEventComment(comment(wrap(eventJSON("e1", "", "op1", "START", "READY", "IN_PROGRESS", `{}`))))
	if err != nil { t.Fatal(err) }
	if e == nil || e.EventID != "e1" || e.CommentID != 1 { t.Fatalf("bad event %#v", e) }
}
func TestParseWorkflowEventIgnoresNormalComment(t *testing.T) {
	e, err := ParseWorkflowEventComment(comment("hello"))
	if err != nil || e != nil { t.Fatalf("%#v %v", e, err) }
}
func TestParseWorkflowEventRejectsUnknownSchema(t *testing.T) {
	_, err := ParseWorkflowEventComment(comment("<!-- iw:workflow-event:v2\n{}\n-->"))
	if WorkflowErrorCode(err) != "UNSUPPORTED_SCHEMA" { t.Fatalf("%v", err) }
}
func TestParseWorkflowEventRejectsMultipleBlocks(t *testing.T) {
	raw := wrap(eventJSON("e1", "", "op1", "START", "READY", "IN_PROGRESS", `{}`))
	_, err := ParseWorkflowEventComment(comment(raw + "\n" + raw))
	if WorkflowErrorCode(err) != "PROTOCOL_ERROR" { t.Fatalf("%v", err) }
}
