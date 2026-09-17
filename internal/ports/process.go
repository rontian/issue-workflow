package ports

import (
	"context"
	"io"
)

type Command struct {
	Name  string
	Args  []string
	Dir   string
	Env   []string
	Stdin []byte
}
type ProcessResult struct {
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Truncated bool
}
type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, Command) (ProcessResult, error)
	RunInteractive(context.Context, Command, io.Reader, io.Writer, io.Writer) error
}
