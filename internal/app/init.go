package app

import (
	"context"
	"fmt"

	"github.com/rontian/issue-workflow/internal/result"
)

type InitOptions struct {
	Doctor                                  DoctorOptions
	DryRun, Interactive, InteractiveTerminal bool
}

type InitAction struct {
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

type InitReport struct {
	Doctor  DoctorReport  `json:"doctor"`
	Actions []InitAction `json:"actions"`
}

type InitService struct{ Doctor *DoctorService }

func (s *InitService) Run(ctx context.Context, o InitOptions) (result.CommandResult, int) {
	if o.Interactive && !o.InteractiveTerminal {
		return result.Failure("init", "INVALID_INVOCATION", "--interactive 只能在交互终端使用", nil, InitReport{Actions: []InitAction{}}), result.ExitInvalid
	}
	dr, _ := s.Doctor.Run(ctx, o.Doctor)
	rep, _ := dr.Data.(DoctorReport)
	acts := []InitAction{}
	if o.Interactive && dr.Error != nil && dr.Error.Code == "AUTH_REQUIRED" {
		host := o.Doctor.Host
		if host == "" && rep.Repository != nil {
			host = rep.Repository.Host
		}
		if o.DryRun {
			acts = append(acts, InitAction{Kind: "gh-auth-login", Status: "PLANNED", Target: host, Message: "dry-run: would run gh auth login"})
		} else if e := s.Doctor.GitHub.Login(ctx, host); e != nil {
			return result.Failure("init", "AUTH_REQUIRED", "gh authentication failed", nil, InitReport{Doctor: rep, Actions: acts}), result.ExitGitHub
		} else {
			acts = append(acts, InitAction{Kind: "gh-auth-login", Status: "DONE", Target: host, Message: "gh auth login completed"})
			dr, _ = s.Doctor.Run(ctx, o.Doctor)
			rep, _ = dr.Data.(DoctorReport)
		}
	}
	data := InitReport{Doctor: rep, Actions: acts}
	if dr.OK {
		r := result.Success("init", data)
		r.Repository = dr.Repository
		r.Issue = dr.Issue
		return r, result.ExitOK
	}
	code := dr.Error.Code
	r := result.Failure("init", code, fmt.Sprintf("环境尚未就绪: %s", dr.Error.Message), dr.Error.Details, data)
	r.Repository = dr.Repository
	r.Issue = dr.Issue
	return r, result.ExitCodeFor(code)
}
