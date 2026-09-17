package process

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rontian/issue-workflow/internal/ports"
)

func TestRunnerArgvNoShell(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip() }
	r := NewRunner()
	got, err := r.Run(context.Background(), ports.Command{Name: "printf", Args: []string{"%s", "$(echo pwned)"}})
	if err != nil { t.Fatal(err) }
	if string(got.Stdout) != "$(echo pwned)" { t.Fatalf("stdout=%q", got.Stdout) }
}
func TestRunnerTimeout(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip() }
	r := NewRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := r.Run(ctx, ports.Command{Name: "sh", Args: []string{"-c", "sleep 2"}})
	if err == nil { t.Fatal("expected timeout") }
}
func TestRunnerOutputBounded(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip() }
	r := &Runner{MaxOutput: 8}
	got, _ := r.Run(context.Background(), ports.Command{Name: "printf", Args: []string{"123456789012"}})
	if !got.Truncated || len(got.Stdout) != 8 { t.Fatalf("got=%q truncated=%v", got.Stdout, got.Truncated) }
}
func TestMergeEnvOverridesOnce(t *testing.T) {
	got := mergeEnv([]string{"A=1", "B=2"}, []string{"A=3"})
	if strings.Join(got, ",") != "A=3,B=2" { t.Fatalf("%v", got) }
}
