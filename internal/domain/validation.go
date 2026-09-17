package domain

import (
	"fmt"
	"strings"
)

type ValidationRequirement struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ValidationRecord struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ObservedHead string `json:"observed_head,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Source       string `json:"source,omitempty"`
}

func ValidationRequirements(c TaskContract) []ValidationRequirement {
	lines := strings.Split(strings.ReplaceAll(c.Validation, "\r\n", "\n"), "\n")
	out := []ValidationRequirement{}
	for _, line := range lines {
		line = strings.TrimSpace(line); if line == "" { continue }
		line = strings.TrimSpace(strings.TrimPrefix(line, "-")); line = strings.Trim(line, "`"); line = strings.TrimSpace(line); if line == "" { continue }
		out = append(out, ValidationRequirement{ID: fmt.Sprintf("v%d", len(out)+1), Text: line})
	}
	return out
}

func ParseValidationRecord(s string) (ValidationRecord, error) {
	p := strings.SplitN(s, "=", 2)
	if len(p) != 2 || strings.TrimSpace(p[0]) == "" { return ValidationRecord{}, fmt.Errorf("validation 必须为 ID=pass|fail|not_run|skipped|manual_required|not_applicable[:summary]") }
	id := strings.TrimSpace(p[0]); rhs := strings.SplitN(strings.TrimSpace(p[1]), ":", 2); st := strings.ToLower(strings.TrimSpace(rhs[0]))
	if st == "skip" { st = "skipped" }
	switch st { case "pass", "fail", "not_run", "skipped", "manual_required", "not_applicable": default: return ValidationRecord{}, fmt.Errorf("unsupported validation status %q", st) }
	r := ValidationRecord{ID: id, Status: st, Source: "agent"}; if len(rhs) == 2 { r.Summary = strings.TrimSpace(rhs[1]) }; return r, nil
}

func ValidationRecordsFromEvents(events []WorkflowEvent) map[string]ValidationRecord {
	out := map[string]ValidationRecord{}
	for _, e := range events {
		raw, ok := e.Data["validation"]; if !ok { continue }
		arr, ok := raw.([]any); if !ok { continue }
		for _, item := range arr {
			m, ok := item.(map[string]any); if !ok { continue }
			id, _ := m["id"].(string); st, _ := m["status"].(string); sum, _ := m["summary"].(string); head, _ := m["observed_head"].(string); source, _ := m["source"].(string)
			if head == "" && e.Git != nil { head = e.Git.Head }
			if source == "" { source = "workflow-event" }
			if id != "" { out[id] = ValidationRecord{ID: id, Status: strings.ToLower(st), ObservedHead: head, Summary: sum, Source: source} }
		}
	}
	return out
}

func MergeValidationRecords(base map[string]ValidationRecord, more []ValidationRecord) map[string]ValidationRecord {
	out := map[string]ValidationRecord{}; for k, v := range base { out[k] = v }; for _, v := range more { out[v.ID] = v }; return out
}

func MissingValidations(req []ValidationRequirement, got map[string]ValidationRecord) []string { return MissingValidationsAtHead(req, got, "") }
func MissingValidationsAtHead(req []ValidationRequirement, got map[string]ValidationRecord, head string) []string {
	out := []string{}
	for _, r := range req {
		v, ok := got[r.ID]
		validStatus := ok && (v.Status == "pass" || v.Status == "not_applicable")
		validHead := head == "" || v.ObservedHead == "" || v.ObservedHead == head
		if !validStatus || !validHead { out = append(out, r.ID) }
	}
	return out
}
