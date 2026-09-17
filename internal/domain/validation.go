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
	ID      string `json:"id"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

func ValidationRequirements(c TaskContract) []ValidationRequirement {
	lines := strings.Split(strings.ReplaceAll(c.Validation, "\r\n", "\n"), "\n")
	out := []ValidationRequirement{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		line = strings.Trim(line, "`")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, ValidationRequirement{ID: fmt.Sprintf("v%d", len(out)+1), Text: line})
	}
	return out
}
func ParseValidationRecord(s string) (ValidationRecord, error) {
	p := strings.SplitN(s, "=", 2)
	if len(p) != 2 || strings.TrimSpace(p[0]) == "" {
		return ValidationRecord{}, fmt.Errorf("validation 必须为 ID=pass|fail|skip[:summary]")
	}
	id := strings.TrimSpace(p[0])
	rhs := strings.SplitN(strings.TrimSpace(p[1]), ":", 2)
	st := strings.ToLower(strings.TrimSpace(rhs[0]))
	if st != "pass" && st != "fail" && st != "skip" {
		return ValidationRecord{}, fmt.Errorf("validation status must be pass/fail/skip")
	}
	r := ValidationRecord{ID: id, Status: st}
	if len(rhs) == 2 {
		r.Summary = strings.TrimSpace(rhs[1])
	}
	return r, nil
}
func ValidationRecordsFromEvents(events []WorkflowEvent) map[string]ValidationRecord {
	out := map[string]ValidationRecord{}
	for _, e := range events {
		raw, ok := e.Data["validation"]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id, _ := m["id"].(string)
			st, _ := m["status"].(string)
			sum, _ := m["summary"].(string)
			if id != "" {
				out[id] = ValidationRecord{ID: id, Status: strings.ToLower(st), Summary: sum}
			}
		}
	}
	return out
}
func MergeValidationRecords(base map[string]ValidationRecord, more []ValidationRecord) map[string]ValidationRecord {
	out := map[string]ValidationRecord{}
	for k, v := range base {
		out[k] = v
	}
	for _, v := range more {
		out[v.ID] = v
	}
	return out
}
func MissingValidations(req []ValidationRequirement, got map[string]ValidationRecord) []string {
	out := []string{}
	for _, r := range req {
		v, ok := got[r.ID]
		if !ok || v.Status != "pass" {
			out = append(out, r.ID)
		}
	}
	return out
}
