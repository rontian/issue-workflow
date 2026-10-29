package gitcli

import (
    "context"
    "net/url"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "github.com/rontian/issue-workflow/internal/process"
)

// Reproduces a single-branch shallow clone whose remote.origin.fetch covers
// only main. Fetching a different ref is not enough for git switch --track:
// the target branch must also be registered in remote.<name>.fetch.
func TestRestoreBranchFromShallowSingleBranchClone(t *testing.T) {
    git, err := exec.LookPath("git")
    if err != nil { t.Skip("git is unavailable") }
    ctx := context.Background()
    root := t.TempDir()
    gitRun := func(dir string, args ...string) string {
        t.Helper()
        cmd := exec.CommandContext(ctx, git, args...)
        cmd.Dir = dir
        out, err := cmd.CombinedOutput()
        if err != nil { t.Fatalf("git %v in %s: %v\n%s", args, dir, err, string(out)) }
        return strings.TrimSpace(string(out))
    }
    bare := filepath.Join(root, "remote.git")
    source := filepath.Join(root, "publisher")
    clone := filepath.Join(root, "receiver")
    if err := os.Mkdir(source, 0700); err != nil { t.Fatal(err) }
    gitRun(root, "init", "--bare", "--initial-branch=main", bare)
    gitRun(source, "init", "--initial-branch=main")
    gitRun(source, "-c", "user.name=iw-test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "main")
    gitRun(source, "remote", "add", "origin", bare)
    gitRun(source, "push", "-u", "origin", "main")
    gitRun(source, "switch", "-c", "portable-case")
    gitRun(source, "-c", "user.name=iw-test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "portable")
    gitRun(source, "push", "-u", "origin", "portable-case")
    head := gitRun(source, "rev-parse", "HEAD")
    fileURL := (&url.URL{Scheme: "file", Path: bare}).String()
    gitRun(root, "clone", "--depth", "1", "--branch", "main", fileURL, clone)
    if spec := gitRun(clone, "config", "--get-all", "remote.origin.fetch"); strings.Contains(spec, "portable-case") {
        t.Fatalf("expected single-branch fetch spec, got %s", spec)
    }

    client := New(process.NewRunner())
    if err := client.FetchRemote(ctx, clone, "origin"); err != nil { t.Fatal(err) }
    if err := client.RestoreBranch(ctx, clone, "origin", "portable-case", head); err != nil {
        t.Fatalf("portable restore from single-branch shallow clone: %v", err)
    }
    if got := gitRun(clone, "rev-parse", "HEAD"); got != head { t.Fatalf("restored %s, expected %s", got, head) }
    if branch := gitRun(clone, "symbolic-ref", "--short", "HEAD"); branch != "portable-case" {
        t.Fatalf("restored branch = %s", branch)
    }
    if upstream := gitRun(clone, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); upstream != "origin/portable-case" {
        t.Fatalf("restored upstream = %s", upstream)
    }
}
