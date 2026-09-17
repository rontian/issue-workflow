package gitcli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
)

var (
	ErrNotRepository = errors.New("not a git repository")
	ErrRemoteMissing = errors.New("git remote not found")
	ErrUnbornRepo    = errors.New("repository has no HEAD commit")
)

type Client struct{ runner ports.Runner }

func New(r ports.Runner) *Client { return &Client{runner: r} }
func (c *Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	x, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := c.runner.Run(x, ports.Command{Name: "git", Args: args, Dir: dir})
	if err != nil { return "", err }
	return strings.TrimSpace(string(res.Stdout)), nil
}
func (c *Client) Version(ctx context.Context) (string, error) { return c.run(ctx, "", "--version") }
func (c *Client) RepositoryRoot(ctx context.Context, dir string) (string, error) {
	o, e := c.run(ctx, dir, "rev-parse", "--show-toplevel")
	if e != nil { return "", fmt.Errorf("%w: %v", ErrNotRepository, e) }
	return o, nil
}
func (c *Client) ResolveRemote(ctx context.Context, dir, requested string) (domain.RepositoryIdentity, string, error) {
	txt, e := c.run(ctx, dir, "remote")
	if e != nil { return domain.RepositoryIdentity{}, "", fmt.Errorf("%w: %v", ErrRemoteMissing, e) }
	rs := []string{}
	for _, r := range strings.Split(txt, "\n") { r = strings.TrimSpace(r); if r != "" { rs = append(rs, r) } }
	name := requested
	if name != "" {
		found := false
		for _, r := range rs { if r == name { found = true; break } }
		if !found { return domain.RepositoryIdentity{}, "", fmt.Errorf("%w: %s", ErrRemoteMissing, name) }
	} else {
		for _, r := range rs { if r == "origin" { name = "origin"; break } }
		if name == "" && len(rs) == 1 { name = rs[0] }
		if name == "" { return domain.RepositoryIdentity{}, "", ErrRemoteMissing }
	}
	raw, e := c.run(ctx, dir, "remote", "get-url", name)
	if e != nil { return domain.RepositoryIdentity{}, "", fmt.Errorf("%w: %s", ErrRemoteMissing, name) }
	id, e := ParseRemoteURL(raw)
	return id, name, e
}
func ParseRemoteURL(raw string) (domain.RepositoryIdentity, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" { return domain.RepositoryIdentity{}, errors.New("empty remote url") }
	if !strings.Contains(raw, "://") && strings.Contains(raw, "@") {
		at := strings.LastIndex(raw, "@"); ci := strings.Index(raw[at+1:], ":")
		if ci >= 0 { ci += at + 1; return identityFromHostPath(raw[at+1:ci], raw[ci+1:]) }
	}
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" { return domain.RepositoryIdentity{}, fmt.Errorf("unsupported git remote: %q", raw) }
	return identityFromHostPath(u.Hostname(), u.Path)
}
func identityFromHostPath(host, path string) (domain.RepositoryIdentity, error) {
	path = strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	p := strings.Split(path, "/")
	if host == "" || len(p) != 2 || p[0] == "" || p[1] == "" { return domain.RepositoryIdentity{}, fmt.Errorf("unsupported GitHub repository path: host=%q path=%q", host, path) }
	return domain.RepositoryIdentity{Host: host, Owner: p[0], Repo: p[1]}, nil
}
func (c *Client) Snapshot(ctx context.Context, dir string) (domain.GitSnapshot, error) {
	root, e := c.RepositoryRoot(ctx, dir)
	if e != nil { return domain.GitSnapshot{}, e }
	head, e := c.run(ctx, root, "rev-parse", "--verify", "HEAD")
	if e != nil { return domain.GitSnapshot{}, fmt.Errorf("%w: %v", ErrUnbornRepo, e) }
	branch, be := c.run(ctx, root, "symbolic-ref", "--quiet", "--short", "HEAD")
	det := be != nil
	x, cancel := context.WithTimeout(ctx, 10*time.Second); defer cancel()
	res, e := c.runner.Run(x, ports.Command{Name: "git", Args: []string{"status", "--porcelain=v2", "-z"}, Dir: root})
	if e != nil { return domain.GitSnapshot{}, e }
	paths, e := ParsePorcelainV2Z(res.Stdout)
	if e != nil { return domain.GitSnapshot{}, e }
	if det { branch = "" }
	return domain.GitSnapshot{RepositoryRoot: root, Branch: branch, Head: head, Detached: det, Dirty: len(paths) > 0, ChangedPaths: paths}, nil
}
func ParsePorcelainV2Z(data []byte) ([]string, error) {
	parts := strings.Split(string(data), "\x00")
	paths := []string{}
	for i := 0; i < len(parts); i++ {
		r := parts[i]; if r == "" { continue }
		switch {
		case strings.HasPrefix(r, "1 "):
			f := strings.SplitN(r, " ", 9); if len(f) != 9 { return nil, errors.New("invalid porcelain v2 type 1") }; paths = append(paths, f[8])
		case strings.HasPrefix(r, "2 "):
			f := strings.SplitN(r, " ", 10); if len(f) != 10 { return nil, errors.New("invalid porcelain v2 type 2") }; paths = append(paths, f[9]); if i+1 < len(parts) { i++ }
		case strings.HasPrefix(r, "u "):
			f := strings.SplitN(r, " ", 11); if len(f) != 11 { return nil, errors.New("invalid porcelain v2 unmerged") }; paths = append(paths, f[10])
		case strings.HasPrefix(r, "? "):
			paths = append(paths, strings.TrimPrefix(r, "? "))
		case strings.HasPrefix(r, "! "), strings.HasPrefix(r, "# "):
		default:
			return nil, fmt.Errorf("unknown porcelain v2 record: %q", r)
		}
	}
	sort.Strings(paths)
	return paths, nil
}
