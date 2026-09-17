package app

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

type projectGitFake struct{ root string }

func (g projectGitFake) Version(context.Context) (string, error)                { return "git", nil }
func (g projectGitFake) RepositoryRoot(context.Context, string) (string, error) { return g.root, nil }
func (g projectGitFake) ResolveRemote(context.Context, string, string) (domain.RepositoryIdentity, string, error) {
	return domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}, "origin", nil
}
func (g projectGitFake) Snapshot(context.Context, string) (domain.GitSnapshot, error) {
	return domain.GitSnapshot{RepositoryRoot: g.root, Branch: "main", Head: "abc", ChangedPaths: []string{}}, nil
}

type projectRunnerFake struct{ available map[string]string }

func (r projectRunnerFake) LookPath(name string) (string, error) {
	if p := r.available[name]; p != "" {
		return p, nil
	}
	return "", errors.New("not found")
}
func (projectRunnerFake) Run(context.Context, ports.Command) (ports.ProcessResult, error) {
	return ports.ProcessResult{}, errors.New("not implemented")
}
func (projectRunnerFake) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error {
	return errors.New("not implemented")
}

func writeIWJSON(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "iw.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findCheckByID(checks []Check, id string) *Check {
	for i := range checks {
		if checks[i].ID == id {
			return &checks[i]
		}
	}
	return nil
}

func TestProjectConfigRejectsCredentialLikeKeys(t *testing.T) {
	for _, key := range []string{"github_token", "credential", "password", "ssh_private_key"} {
		t.Run(key, func(t *testing.T) {
			_, err := parseProjectConfig([]byte(`{"schema":"iw.project/v1","` + key + `":"do-not-store"}`))
			if err == nil || !strings.Contains(err.Error(), "must not contain credential/private configuration key") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestProjectConfigRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "absolute docs path", body: `{"schema":"iw.project/v1","docs":["/private/readme.md"]}`, want: "repository-relative"},
		{name: "docs traversal", body: `{"schema":"iw.project/v1","docs":["../private/readme.md"]}`, want: "repository-relative"},
		{name: "executable path", body: `{"schema":"iw.project/v1","environment":{"executables":["./bin/tool"]}}`, want: "command name, not a path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseProjectConfig([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestProjectContractValid(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeIWJSON(t, root, `{
  "schema": "iw.project/v1",
  "project_id": "core",
  "default_branch": "main",
  "docs": ["README.md"],
  "environment": {"executables": ["go"]},
  "policy": {"require_review": true}
}`)
	s := &ProjectService{Git: projectGitFake{root: root}, Runner: projectRunnerFake{available: map[string]string{"go": "/usr/bin/go"}}}
	rep, err := s.Read(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Configured || rep.Config.ProjectID != "core" || rep.Config.DefaultBranch != "main" || !rep.Config.Policy.RequireReview {
		t.Fatalf("report=%#v", rep)
	}
	if len(rep.Docs) != 1 || !rep.Docs[0].Exists {
		t.Fatalf("docs=%#v", rep.Docs)
	}
	if len(rep.Environment) != 1 || !rep.Environment[0].Available || rep.Environment[0].Resolved != "/usr/bin/go" {
		t.Fatalf("environment=%#v", rep.Environment)
	}
}

func TestProjectDoctorFailsWhenDeclaredDocMissing(t *testing.T) {
	root := t.TempDir()
	writeIWJSON(t, root, `{"schema":"iw.project/v1","docs":["docs/missing.md"]}`)
	s := &ProjectService{Git: projectGitFake{root: root}, Runner: projectRunnerFake{available: map[string]string{}}}
	base := result.Success("doctor", DoctorReport{Checks: []Check{}})
	r, code := s.AugmentDoctor(context.Background(), root, base, result.ExitOK)
	if code != result.ExitEnvironment || r.OK || r.Error == nil || r.Error.Code != "ENVIRONMENT_INVALID" {
		t.Fatalf("code=%d result=%#v", code, r)
	}
	rep := r.Data.(DoctorReport)
	check := findCheckByID(rep.Checks, "project.doc.001")
	if check == nil || check.Status != CheckFail || !check.Required {
		t.Fatalf("checks=%#v", rep.Checks)
	}
}

func TestProjectDoctorFailsWhenDeclaredExecutableMissing(t *testing.T) {
	root := t.TempDir()
	writeIWJSON(t, root, `{"schema":"iw.project/v1","environment":{"executables":["go"]}}`)
	s := &ProjectService{Git: projectGitFake{root: root}, Runner: projectRunnerFake{available: map[string]string{}}}
	base := result.Success("doctor", DoctorReport{Checks: []Check{}})
	r, code := s.AugmentDoctor(context.Background(), root, base, result.ExitOK)
	if code != result.ExitEnvironment || r.OK || r.Error == nil || r.Error.Code != "DEPENDENCY_MISSING" {
		t.Fatalf("code=%d result=%#v", code, r)
	}
	rep := r.Data.(DoctorReport)
	check := findCheckByID(rep.Checks, "project.executable.go")
	if check == nil || check.Status != CheckFail || !check.Required {
		t.Fatalf("checks=%#v", rep.Checks)
	}
}
