package app

import (
	"context"
	"errors"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

type portableGitFake struct {
	repo           domain.RepositoryIdentity
	snap           domain.GitSnapshot
	remoteHead     string
	commitExists   bool
	fetchErr       error
	restoreErr     error
	fetchCalls     int
	restoreCalls   int
	restoredRemote string
	restoredBranch string
	restoredHead   string
}

func (g *portableGitFake) Version(context.Context) (string, error) { return "git", nil }
func (g *portableGitFake) RepositoryRoot(context.Context, string) (string, error) {
	return g.snap.RepositoryRoot, nil
}
func (g *portableGitFake) ResolveRemote(context.Context, string, string) (domain.RepositoryIdentity, string, error) {
	return g.repo, "origin", nil
}
func (g *portableGitFake) Snapshot(context.Context, string) (domain.GitSnapshot, error) {
	return g.snap, nil
}
func (g *portableGitFake) RemoteBranchHead(context.Context, string, string, string) (string, error) {
	if g.remoteHead == "" {
		return "", errors.New("remote branch unavailable")
	}
	return g.remoteHead, nil
}
func (g *portableGitFake) FetchRemote(context.Context, string, string) error {
	g.fetchCalls++
	return g.fetchErr
}
func (g *portableGitFake) CommitExists(context.Context, string, string) bool { return g.commitExists }
func (g *portableGitFake) RestoreBranch(_ context.Context, _ string, remote, branch, head string) error {
	g.restoreCalls++
	g.restoredRemote, g.restoredBranch, g.restoredHead = remote, branch, head
	return g.restoreErr
}

func portableStateService(t *testing.T) (*WorkflowService, *stateGH, *portableGitFake) {
	t.Helper()
	repo := domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}
	g := &stateGH{
		repo: domain.GitHubRepository{NameWithOwner: "o/r", HasIssuesEnabled: true},
		issue: domain.GitHubIssueDetails{
			Number: 7,
			State:  "OPEN",
			Title:  "Task",
			Body:   testContract(t, false),
		},
	}
	git := &portableGitFake{
		repo:         repo,
		snap:         domain.GitSnapshot{RepositoryRoot: "/repo", Branch: "main", Head: "abc", Dirty: false, ChangedPaths: []string{}},
		remoteHead:   "abc",
		commitExists: true,
	}
	return &WorkflowService{Git: git, GitHub: g}, g, git
}

func persistPortableHandoff(t *testing.T, s *WorkflowService) {
	t.Helper()
	if r, c := s.Start(context.Background(), mo("portable-start")); c != result.ExitOK || !r.OK {
		t.Fatalf("start %d %#v", c, r)
	}
	h := mo("portable-handoff")
	h.Summary = "ready for another machine"
	h.Next = "restore and continue"
	h.Portable = true
	if r, c := s.Handoff(context.Background(), h); c != result.ExitOK || !r.OK {
		t.Fatalf("handoff %d %#v", c, r)
	}
}

func TestPortableHandoffFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*portableGitFake)
	}{
		{name: "dirty worktree", mutate: func(g *portableGitFake) {
			g.snap.Dirty = true
			g.snap.ChangedPaths = []string{"dirty.txt"}
		}},
		{name: "detached HEAD", mutate: func(g *portableGitFake) {
			g.snap.Detached = true
			g.snap.Branch = ""
		}},
		{name: "remote HEAD mismatch", mutate: func(g *portableGitFake) {
			g.remoteHead = "def"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, gh, git := portableStateService(t)
			if r, c := s.Start(context.Background(), mo("start")); c != result.ExitOK || !r.OK {
				t.Fatalf("start %d %#v", c, r)
			}
			tc.mutate(git)
			h := mo("handoff")
			h.Summary = "handoff"
			h.Next = "continue elsewhere"
			h.Portable = true
			r, c := s.Handoff(context.Background(), h)
			if c != result.ExitConflict || r.Error == nil || r.Error.Code != "STALE_GIT_STATE" {
				t.Fatalf("handoff %d %#v", c, r)
			}
			if gh.appendCalls != 1 {
				t.Fatalf("failed portable handoff must not append event; appendCalls=%d", gh.appendCalls)
			}
		})
	}
}

func TestPortableHandoffPersistsExactRestoreCoordinates(t *testing.T) {
	s, _, _ := portableStateService(t)
	persistPortableHandoff(t, s)
	r, c := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if c != result.ExitOK || !r.OK {
		t.Fatalf("status %d %#v", c, r)
	}
	p := r.Data.(WorkflowReport).Aggregate.PortableHandoff
	if p == nil || p.Remote != "origin" || p.Branch != "main" || p.Head != "abc" {
		t.Fatalf("portable handoff=%#v", p)
	}
}

func TestRestoreDefaultsToPlanOnly(t *testing.T) {
	s, _, git := portableStateService(t)
	persistPortableHandoff(t, s)
	git.snap.Branch, git.snap.Head = "other", "def"
	rs := &RestoreService{Workflow: s, Git: git}
	r, c := rs.Run(context.Background(), RestoreOptions{WorkflowOptions: WorkflowOptions{CWD: "/repo", Issue: 7}})
	if c != result.ExitOK || !r.OK {
		t.Fatalf("restore plan %d %#v", c, r)
	}
	rep := r.Data.(RestoreReport)
	if rep.Apply || !rep.RemoteVerified || rep.Applied || git.fetchCalls != 0 || git.restoreCalls != 0 {
		t.Fatalf("plan=%#v fetch=%d restore=%d", rep, git.fetchCalls, git.restoreCalls)
	}
}

func TestRestoreDirtyWorktreeFailsClosed(t *testing.T) {
	s, _, git := portableStateService(t)
	persistPortableHandoff(t, s)
	git.snap.Dirty = true
	git.snap.ChangedPaths = []string{"dirty.txt"}
	rs := &RestoreService{Workflow: s, Git: git}
	r, c := rs.Run(context.Background(), RestoreOptions{WorkflowOptions: WorkflowOptions{CWD: "/repo", Issue: 7}, Apply: true})
	if c != result.ExitConflict || r.Error == nil || r.Error.Code != "STALE_GIT_STATE" {
		t.Fatalf("restore %d %#v", c, r)
	}
	if git.fetchCalls != 0 || git.restoreCalls != 0 {
		t.Fatalf("dirty restore must have no mutation: fetch=%d restore=%d", git.fetchCalls, git.restoreCalls)
	}
}

func TestRestoreRemoteBranchDriftFailsClosed(t *testing.T) {
	s, _, git := portableStateService(t)
	persistPortableHandoff(t, s)
	git.snap.Branch, git.snap.Head = "other", "def"
	git.remoteHead = "drifted"
	rs := &RestoreService{Workflow: s, Git: git}
	r, c := rs.Run(context.Background(), RestoreOptions{WorkflowOptions: WorkflowOptions{CWD: "/repo", Issue: 7}, Apply: true})
	if c != result.ExitConflict || r.Error == nil || r.Error.Code != "RESTORE_CONFLICT" {
		t.Fatalf("restore %d %#v", c, r)
	}
	if git.fetchCalls != 0 || git.restoreCalls != 0 {
		t.Fatalf("remote drift must fail before mutation: fetch=%d restore=%d", git.fetchCalls, git.restoreCalls)
	}
}

func TestRestoreApplyFetchesThenUsesSafeBranchRestore(t *testing.T) {
	s, _, git := portableStateService(t)
	persistPortableHandoff(t, s)
	git.snap.Branch, git.snap.Head = "other", "def"
	rs := &RestoreService{Workflow: s, Git: git}
	r, c := rs.Run(context.Background(), RestoreOptions{WorkflowOptions: WorkflowOptions{CWD: "/repo", Issue: 7}, Apply: true})
	if c != result.ExitOK || !r.OK {
		t.Fatalf("restore apply %d %#v", c, r)
	}
	rep := r.Data.(RestoreReport)
	if !rep.Apply || !rep.RemoteVerified || !rep.Applied {
		t.Fatalf("restore report=%#v", rep)
	}
	if git.fetchCalls != 1 || git.restoreCalls != 1 {
		t.Fatalf("fetch=%d restore=%d", git.fetchCalls, git.restoreCalls)
	}
	if git.restoredRemote != "origin" || git.restoredBranch != "main" || git.restoredHead != "abc" {
		t.Fatalf("restore coordinates=%s/%s@%s", git.restoredRemote, git.restoredBranch, git.restoredHead)
	}
}
