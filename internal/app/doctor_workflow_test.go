package app

import (
	"context"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func TestAugmentDoctorIssueAddsProtocolPass(t *testing.T) {
	body := contractBody()
	s := newService(body, []domain.GitHubComment{workflowComment(t, domain.ContractDigest(body))})
	baseReport := DoctorReport{Checks: []Check{{ID: "dependency.gh", Category: "dependency", Status: CheckPass, Required: true, Message: "ok"}}, Summary: CheckSummary{Pass: 1}}
	base := result.Success("doctor", baseReport)
	base.Repository = "o/r"
	n := 7
	base.Issue = &n
	r, code := s.AugmentDoctorIssue(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7}, base, 0)
	if code != 0 || !r.OK { t.Fatalf("%d %#v", code, r) }
	report := r.Data.(DoctorReport)
	found := false
	for _, c := range report.Checks { if c.ID == "workflow.protocol" && c.Status == CheckPass { found = true } }
	if !found { t.Fatalf("%#v", report.Checks) }
}

func TestAugmentDoctorIssueSkipsWhenBaseFailed(t *testing.T) {
	s := &WorkflowService{}
	base := result.Failure("doctor", "DEPENDENCY_MISSING", "gh missing", nil, DoctorReport{Checks: []Check{{ID: "dependency.gh", Category: "dependency", Status: CheckFail, Required: true, Message: "missing"}}, Summary: CheckSummary{Fail: 1}})
	r, code := s.AugmentDoctorIssue(context.Background(), WorkflowOptions{Issue: 7}, base, 3)
	if code != 3 || r.OK { t.Fatalf("%d %#v", code, r) }
	report := r.Data.(DoctorReport)
	found := false
	for _, c := range report.Checks { if c.ID == "workflow.protocol" && c.Status == CheckSkip { found = true } }
	if !found { t.Fatalf("%#v", report.Checks) }
}
