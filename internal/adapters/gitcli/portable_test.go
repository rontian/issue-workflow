package gitcli

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rontian/issue-workflow/internal/ports"
)

type scriptedPortableRunner struct {
	commands  []ports.Command
	responses []ports.ProcessResult
	errors    []error
}

func (r *scriptedPortableRunner) LookPath(name string) (string, error) { return name, nil }
func (r *scriptedPortableRunner) Run(_ context.Context, cmd ports.Command) (ports.ProcessResult, error) {
	i := len(r.commands)
	r.commands = append(r.commands, cmd)
	if i >= len(r.responses) {
		return ports.ProcessResult{}, fmt.Errorf("unexpected command: %v", cmd.Args)
	}
	var err error
	if i < len(r.errors) {
		err = r.errors[i]
	}
	return r.responses[i], err
}
func (r *scriptedPortableRunner) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error {
	return nil
}

func TestRestoreBranchUsesOnlySafeFastForwardOperations(t *testing.T) {
	r := &scriptedPortableRunner{responses: []ports.ProcessResult{
		{Stdout: []byte("abc\trefs/heads/main\n")},
		{Stdout: []byte("old\n")},
		{},
		{},
		{Stdout: []byte("abc\n")},
	}}
	c := New(r)
	if err := c.RestoreBranch(context.Background(), "/repo", "origin", "main", "abc"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"ls-remote", "--heads", "origin", "refs/heads/main"},
		{"show-ref", "--verify", "--hash", "refs/heads/main"},
		{"switch", "main"},
		{"merge", "--ff-only", "abc"},
		{"rev-parse", "--verify", "HEAD"},
	}
	if len(r.commands) != len(want) {
		t.Fatalf("commands=%v", r.commands)
	}
	for i, cmd := range r.commands {
		if cmd.Name != "git" || cmd.Dir != "/repo" || !reflect.DeepEqual(cmd.Args, want[i]) {
			t.Fatalf("command %d = %#v, want args %v", i, cmd, want[i])
		}
		joined := strings.Join(cmd.Args, " ")
		if strings.Contains(joined, "reset") || strings.Contains(joined, "rebase") || strings.Contains(joined, "push") || strings.Contains(joined, "--force") {
			t.Fatalf("unsafe restore command: %q", joined)
		}
	}
}

func TestFetchRemoteRetrievesNonDefaultBranchFromShallowClone(t *testing.T){
    runner:=&scriptedPortableRunner{responses:[]ports.ProcessResult{{}}}
    client:=New(runner)
    if err:=client.FetchRemote(context.Background(),"/repo","origin");err!=nil{t.Fatalf("fetch: %v",err)}
    want:=[]string{"fetch","--no-tags","--prune","origin","+refs/heads/*:refs/remotes/origin/*"}
    if len(runner.commands)!=1||!reflect.DeepEqual(runner.commands[0].Args,want){
        t.Fatalf("fetch must obtain all remote branch refs even when clone fetchspec is single-branch: %#v",runner.commands)
    }
}
