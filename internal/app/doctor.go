package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
	"github.com/rontian/issue-workflow/internal/version"
)

type DoctorOptions struct {
	CWD, Repo, Remote, Host string
	Global                  bool
	Issue                   int
}

type DoctorService struct {
	Runner ports.Runner
	Git    ports.GitPort
	GitHub ports.GitHubPort
}

func (s *DoctorService) Run(ctx context.Context, o DoctorOptions) (result.CommandResult, int) {
	cs := []Check{{ID: "dependency.iw", Category: "dependency", Status: CheckPass, Required: true, Message: "iw " + version.Version}}
	add := func(c Check) { cs = append(cs, c) }
	gitOK := true
	ghOK := true
	if s.Runner == nil {
		add(Check{ID: "dependency.runner", Category: "dependency", Status: CheckFail, Required: true, Message: "process runner unavailable"})
		gitOK = false
		ghOK = false
	} else {
		if _, e := s.Runner.LookPath("git"); e != nil {
			add(Check{ID: "dependency.git", Category: "dependency", Status: CheckFail, Required: true, Message: "git command not found", Next: "安装 Git"})
			gitOK = false
		} else if v, e := s.Git.Version(ctx); e != nil {
			add(Check{ID: "dependency.git", Category: "dependency", Status: CheckFail, Required: true, Message: "git 不可执行"})
			gitOK = false
		} else {
			add(Check{ID: "dependency.git", Category: "dependency", Status: CheckPass, Required: true, Message: v})
		}
		if _, e := s.Runner.LookPath("gh"); e != nil {
			add(Check{ID: "dependency.gh", Category: "dependency", Status: CheckFail, Required: true, Message: "gh command not found", Next: "安装 GitHub CLI: https://cli.github.com/"})
			ghOK = false
		} else if v, e := s.GitHub.Version(ctx); e != nil {
			add(Check{ID: "dependency.gh", Category: "dependency", Status: CheckFail, Required: true, Message: "gh 不可执行"})
			ghOK = false
		} else {
			add(Check{ID: "dependency.gh", Category: "dependency", Status: CheckPass, Required: true, Message: v})
		}
	}
	if o.Global {
		sortChecks(cs)
		rep := DoctorReport{Checks: cs, Summary: summarize(cs)}
		if f := firstRequiredFailure(cs); f != nil {
			return doctorFailure(rep, f)
		}
		return result.Success("doctor", rep), result.ExitOK
	}
	var repo domain.RepositoryIdentity
	if gitOK {
		root, e := s.Git.RepositoryRoot(ctx, o.CWD)
		if e != nil {
			add(Check{ID: "git.repository", Category: "git", Status: CheckFail, Required: true, Message: "当前目录不是 Git repository"})
			gitOK = false
		} else {
			add(Check{ID: "git.repository", Category: "git", Status: CheckPass, Required: true, Message: root})
			rid, _, e := s.Git.ResolveRemote(ctx, root, o.Remote)
			if e != nil {
				add(Check{ID: "git.remote", Category: "git", Status: CheckFail, Required: true, Message: "无法解析 Git remote"})
				gitOK = false
			} else if o.Repo != "" && !strings.EqualFold(rid.FullName(), o.Repo) {
				add(Check{ID: "git.remote", Category: "git", Status: CheckFail, Required: true, Message: "--repo 与 Git remote 不一致"})
				gitOK = false
			} else {
				if o.Repo != "" {
					p := strings.Split(o.Repo, "/")
					rid.Owner, rid.Repo = p[0], p[1]
				}
				if o.Host != "" {
					rid.Host = o.Host
				}
				repo = rid
				add(Check{ID: "git.remote", Category: "git", Status: CheckPass, Required: true, Message: repo.Host + "/" + repo.FullName()})
				if snap, e := s.Git.Snapshot(ctx, root); e != nil {
					add(Check{ID: "git.snapshot", Category: "git", Status: CheckFail, Required: true, Message: "无法读取 Git snapshot"})
					gitOK = false
				} else {
					add(Check{ID: "git.snapshot", Category: "git", Status: CheckPass, Required: true, Message: fmt.Sprintf("%s %s dirty=%t", snap.Branch, snap.Head, snap.Dirty)})
				}
			}
		}
	}
	if ghOK && gitOK {
		if e := s.GitHub.Authenticated(ctx, repo.Host); e != nil {
			add(Check{ID: "github.auth", Category: "github", Status: CheckFail, Required: true, Message: "gh 未认证 host", Next: "运行 gh auth login --hostname " + repo.Host})
			ghOK = false
		} else {
			add(Check{ID: "github.auth", Category: "github", Status: CheckPass, Required: true, Message: "authenticated: " + repo.Host})
		}
		if ghOK {
			remote, e := s.GitHub.Repository(ctx, repo)
			if e != nil {
				add(Check{ID: "github.repository", Category: "github", Status: CheckFail, Required: true, Message: "GitHub repository 不可访问"})
				ghOK = false
			} else if !strings.EqualFold(remote.NameWithOwner, repo.FullName()) {
				add(Check{ID: "github.repository", Category: "github", Status: CheckFail, Required: true, Message: "repository identity mismatch"})
				ghOK = false
			} else if !remote.HasIssuesEnabled {
				add(Check{ID: "github.repository", Category: "github", Status: CheckFail, Required: true, Message: "repository Issues 未启用"})
				ghOK = false
			} else {
				add(Check{ID: "github.repository", Category: "github", Status: CheckPass, Required: true, Message: remote.NameWithOwner})
			}
		}
		if ghOK && o.Issue > 0 {
			iss, e := s.GitHub.Issue(ctx, repo, o.Issue)
			if e != nil {
				add(Check{ID: "github.issue", Category: "github", Status: CheckFail, Required: true, Message: "Issue 不可读取"})
			} else {
				add(Check{ID: "github.issue", Category: "github", Status: CheckPass, Required: true, Message: fmt.Sprintf("#%d %s", iss.Number, iss.Title)})
			}
		}
	}
	sortChecks(cs)
	rep := DoctorReport{Checks: cs, Summary: summarize(cs)}
	if repo.FullName() != "" {
		rep.Repository = &repo
	}
	if f := firstRequiredFailure(cs); f != nil {
		return doctorFailure(rep, f)
	}
	r := result.Success("doctor", rep)
	if rep.Repository != nil {
		r.Repository = rep.Repository.FullName()
	}
	if o.Issue > 0 {
		n := o.Issue
		r.Issue = &n
	}
	return r, result.ExitOK
}

func doctorFailure(rep DoctorReport, f *Check) (result.CommandResult, int) {
	code := "ENVIRONMENT_INVALID"
	if f.ID == "dependency.git" || f.ID == "dependency.gh" {
		code = "DEPENDENCY_MISSING"
	} else if f.ID == "github.auth" {
		code = "AUTH_REQUIRED"
	} else if f.ID == "github.repository" {
		code = "REPOSITORY_NOT_FOUND"
	} else if f.ID == "github.issue" {
		code = "ISSUE_NOT_FOUND"
	} else if f.ID == "git.repository" {
		code = "NOT_A_GIT_REPOSITORY"
	} else if f.ID == "git.remote" {
		code = "REMOTE_NOT_FOUND"
	}
	r := result.Failure("doctor", code, f.Message, nil, rep)
	if rep.Repository != nil {
		r.Repository = rep.Repository.FullName()
	}
	return r, result.ExitCodeFor(code)
}
