package main

import (
	"context"
	"os"

	"github.com/rontian/issue-workflow/internal/adapters/ghcli"
	"github.com/rontian/issue-workflow/internal/adapters/gitcli"
	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/cli"
	processrunner "github.com/rontian/issue-workflow/internal/process"
)

func main() {
	runner := processrunner.NewRunner()
	git := gitcli.New(runner)
	github := ghcli.New(runner, os.Stdin, os.Stdout, os.Stderr)
	doctor := &app.DoctorService{Runner: runner, Git: git, GitHub: github}
	initializer := &app.InitService{Doctor: doctor}
	workflow := &app.WorkflowService{Runner: runner, Git: git, GitHub: github}
	command := &cli.CLI{Doctor: doctor, Init: initializer, Workflow: workflow, Stdout: os.Stdout, Stderr: os.Stderr, Getwd: os.Getwd, IsTerminal: stdinIsTerminal}
	os.Exit(command.Run(context.Background(), os.Args[1:]))
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
