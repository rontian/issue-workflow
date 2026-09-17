package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

const ProjectSchema = "iw.project/v1"

type ProjectEnvironment struct {
	Executables []string `json:"executables,omitempty"`
}

type ProjectPolicy struct {
	RequireReview bool `json:"require_review,omitempty"`
}

type ProjectConfig struct {
	Schema        string             `json:"schema"`
	ProjectID     string             `json:"project_id,omitempty"`
	DefaultBranch string             `json:"default_branch,omitempty"`
	Docs          []string           `json:"docs,omitempty"`
	Environment   ProjectEnvironment `json:"environment,omitempty"`
	Policy        ProjectPolicy      `json:"policy,omitempty"`
}

type ProjectDocStatus struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type ProjectExecutableStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Resolved  string `json:"resolved,omitempty"`
}

type ProjectReport struct {
	Schema      string                    `json:"schema"`
	Root        string                    `json:"root"`
	Configured  bool                      `json:"configured"`
	ConfigPath  string                    `json:"config_path,omitempty"`
	Config      ProjectConfig             `json:"config"`
	Docs        []ProjectDocStatus        `json:"docs"`
	Environment []ProjectExecutableStatus `json:"environment"`
}

type ProjectService struct {
	Runner ports.Runner
	Git    ports.GitPort
}

func (s *ProjectService) Status(ctx context.Context, cwd string) (result.CommandResult, int) {
	rep, err := s.Read(ctx, cwd)
	if err != nil {
		code := "PROJECT_CONFIG_INVALID"
		if strings.Contains(err.Error(), "Git repository") {
			code = "NOT_A_GIT_REPOSITORY"
		}
		return result.Failure("project status", code, err.Error(), nil, nil), result.ExitCodeFor(code)
	}
	return result.Success("project status", rep), result.ExitOK
}

func (s *ProjectService) Read(ctx context.Context, cwd string) (ProjectReport, error) {
	if s.Git == nil {
		return ProjectReport{}, fmt.Errorf("Git repository service unavailable")
	}
	root, err := s.Git.RepositoryRoot(ctx, cwd)
	if err != nil {
		return ProjectReport{}, fmt.Errorf("Git repository unavailable: %w", err)
	}
	rep := ProjectReport{Schema: ProjectSchema, Root: root, Docs: []ProjectDocStatus{}, Environment: []ProjectExecutableStatus{}, Config: ProjectConfig{Schema: ProjectSchema}}
	path := filepath.Join(root, "iw.json")
	b, err := os.ReadFile(path)
	if err == nil {
		rep.Configured = true
		rep.ConfigPath = "iw.json"
		cfg, e := parseProjectConfig(b)
		if e != nil {
			return ProjectReport{}, e
		}
		rep.Config = cfg
	} else if !os.IsNotExist(err) {
		return ProjectReport{}, fmt.Errorf("read iw.json: %w", err)
	}
	for _, p := range rep.Config.Docs {
		_, e := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		rep.Docs = append(rep.Docs, ProjectDocStatus{Path: p, Exists: e == nil})
	}
	for _, name := range rep.Config.Environment.Executables {
		st := ProjectExecutableStatus{Name: name}
		if s.Runner != nil {
			if p, e := s.Runner.LookPath(name); e == nil {
				st.Available, st.Resolved = true, p
			}
		}
		rep.Environment = append(rep.Environment, st)
	}
	return rep, nil
}

func parseProjectConfig(b []byte) (ProjectConfig, error) {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return ProjectConfig{}, fmt.Errorf("invalid iw.json: %w", err)
	}
	if err := rejectPrivateConfig(raw, ""); err != nil {
		return ProjectConfig{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var cfg ProjectConfig
	if err := dec.Decode(&cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("invalid iw.json fields: %w", err)
	}
	if cfg.Schema == "" {
		cfg.Schema = ProjectSchema
	}
	if cfg.Schema != ProjectSchema {
		return ProjectConfig{}, fmt.Errorf("unsupported project schema %q", cfg.Schema)
	}
	for _, p := range cfg.Docs {
		if filepath.IsAbs(p) || p == "" || strings.HasPrefix(filepath.Clean(p), "..") {
			return ProjectConfig{}, fmt.Errorf("project docs path must be repository-relative: %q", p)
		}
	}
	for _, name := range cfg.Environment.Executables {
		if name == "" || strings.ContainsAny(name, `/\\`) {
			return ProjectConfig{}, fmt.Errorf("environment executable must be a command name, not a path: %q", name)
		}
	}
	return cfg, nil
}

func rejectPrivateConfig(v any, prefix string) error {
	forbidden := []string{"token", "secret", "credential", "password", "private_key", "privatekey", "ssh_key", "apikey", "api_key"}
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			lk := strings.ToLower(k)
			for _, f := range forbidden {
				if strings.Contains(lk, f) {
					return fmt.Errorf("iw.json must not contain credential/private configuration key %q", joinConfigKey(prefix, k))
				}
			}
			if err := rejectPrivateConfig(vv, joinConfigKey(prefix, k)); err != nil {
				return err
			}
		}
	case []any:
		for i, vv := range x {
			if err := rejectPrivateConfig(vv, fmt.Sprintf("%s[%d]", prefix, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func joinConfigKey(prefix, key string) string {
	if prefix == "" { return key }
	return prefix + "." + key
}
