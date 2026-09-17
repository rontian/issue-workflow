package app

import (
	"context"
	"testing"

	"github.com/rontian/issue-workflow/internal/result"
)

func TestTasksMarksClosedActiveIssueUnhealthy(t *testing.T) {
	s, g, _ := stateService(t, false)
	g.issue.State = "CLOSED"
	tasks := &TaskService{Workflow: s}
	r, code := tasks.Tasks(context.Background(), WorkflowOptions{CWD: "/repo"})
	if code != result.ExitOK || !r.OK {
		t.Fatalf("tasks %d %#v", code, r)
	}
	report := r.Data.(TasksReport)
	if len(report.Tasks) != 1 {
		t.Fatalf("tasks=%d", len(report.Tasks))
	}
	task := report.Tasks[0]
	if task.Ready {
		t.Fatalf("closed issue must not be ready: %#v", task)
	}
	found := false
	for _, health := range task.Health {
		if health == "GITHUB_ISSUE_CLOSED" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("health=%v", task.Health)
	}

	next, code := tasks.Next(context.Background(), WorkflowOptions{CWD: "/repo"})
	if code != result.ExitOK || !next.OK {
		t.Fatalf("next %d %#v", code, next)
	}
	if next.Data.(NextReport).Task != nil || next.Issue != nil {
		t.Fatalf("closed unhealthy task must not be selected: %#v", next)
	}
}

func TestNextSetsTopLevelIssueForSelectedTask(t *testing.T) {
	s, _, _ := stateService(t, false)
	tasks := &TaskService{Workflow: s}
	r, code := tasks.Next(context.Background(), WorkflowOptions{CWD: "/repo"})
	if code != result.ExitOK || !r.OK {
		t.Fatalf("next %d %#v", code, r)
	}
	if r.Issue == nil || *r.Issue != 7 {
		t.Fatalf("top-level issue=%v", r.Issue)
	}
	if r.Data.(NextReport).Task == nil || r.Data.(NextReport).Task.Issue != 7 {
		t.Fatalf("selected=%#v", r.Data)
	}
}
