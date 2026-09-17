package process

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rontian/issue-workflow/internal/ports"
)

const defaultMaxOutput = 4 << 20

type Runner struct{ MaxOutput int }

func NewRunner() *Runner                               { return &Runner{MaxOutput: defaultMaxOutput} }
func (r *Runner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (r *Runner) Run(ctx context.Context, c ports.Command) (ports.ProcessResult, error) {
	max := r.MaxOutput
	if max <= 0 { max = defaultMaxOutput }
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 { cmd.Env = mergeEnv(os.Environ(), c.Env) }
	if c.Stdin != nil { cmd.Stdin = bytes.NewReader(c.Stdin) }
	out := &limitedBuffer{limit: max}
	errOut := &limitedBuffer{limit: max}
	cmd.Stdout, cmd.Stderr = out, errOut
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) { code = ee.ExitCode() } else { code = -1 }
	}
	return ports.ProcessResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), ExitCode: code, Truncated: out.truncated || errOut.truncated}, err
}

func (r *Runner) RunInteractive(ctx context.Context, c ports.Command, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 { cmd.Env = mergeEnv(os.Environ(), c.Env) }
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run()
}

type limitedBuffer struct { buf bytes.Buffer; limit int; truncated bool }
func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remain := b.limit - b.buf.Len()
	if remain > 0 {
		if len(p) > remain { _, _ = b.buf.Write(p[:remain]); b.truncated = true } else { _, _ = b.buf.Write(p) }
	} else if len(p) > 0 { b.truncated = true }
	return n, nil
}
func (b *limitedBuffer) Bytes() []byte { return append([]byte(nil), b.buf.Bytes()...) }

func mergeEnv(base, override []string) []string {
	vals := map[string]string{}
	order := []string{}
	apply := func(items []string) {
		for _, e := range items {
			k := e
			if i := strings.IndexByte(e, '='); i >= 0 { k = e[:i] }
			if _, ok := vals[k]; !ok { order = append(order, k) }
			vals[k] = e
		}
	}
	apply(base); apply(override)
	out := make([]string, 0, len(order))
	for _, k := range order { out = append(out, vals[k]) }
	return out
}
