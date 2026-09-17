package result

import "strings"

const (
	Schema                 = "iw.command-result/v2"
	LegacySchema           = "iw.command-result/v1"
	MachineSemanticsSchema = "iw.machine-semantics/v1"
)

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
type MachineSignal struct {
	Code     string `json:"code"`
	Blocking bool   `json:"blocking,omitempty"`
	Message  string `json:"message,omitempty"`
}
type MachineAction struct {
	ID         string `json:"id"`
	Command    string `json:"command,omitempty"`
	Issue      *int   `json:"issue,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
}
type MachineSemantics struct {
	Schema         string          `json:"schema"`
	Warnings       []MachineSignal `json:"warnings"`
	Constraints    []MachineSignal `json:"constraints"`
	AllowedActions []string        `json:"allowed_actions"`
	NextActions    []MachineAction `json:"next_actions"`
}

type CommandResult struct {
	Schema           string           `json:"schema"`
	OK               bool             `json:"ok"`
	Command          string           `json:"command"`
	CanonicalCommand string           `json:"canonical_command,omitempty"`
	Repository       string           `json:"repository,omitempty"`
	Issue            *int             `json:"issue,omitempty"`
	State            string           `json:"state,omitempty"`
	Lifecycle        string           `json:"lifecycle,omitempty"`
	Warnings         []string         `json:"warnings"`
	Constraints      []string         `json:"constraints"`
	NextActions      []string         `json:"next_actions"`
	Machine          MachineSemantics `json:"machine"`
	Data             any              `json:"data"`
	Error            *Error           `json:"error"`
}

func Success(c string, d any) CommandResult {
	return CommandResult{Schema: Schema, OK: true, Command: c, CanonicalCommand: c, Warnings: []string{}, Constraints: []string{}, NextActions: []string{}, Machine: newMachine(), Data: d}
}
func Failure(c, code, msg string, details map[string]any, d any) CommandResult {
	return CommandResult{Schema: Schema, OK: false, Command: c, CanonicalCommand: c, Warnings: []string{}, Constraints: []string{}, NextActions: []string{}, Machine: newMachine(), Data: d, Error: &Error{Code: code, Message: msg, Details: details}}
}
func newMachine() MachineSemantics {
	return MachineSemantics{Schema: MachineSemanticsSchema, Warnings: []MachineSignal{}, Constraints: []MachineSignal{}, AllowedActions: []string{}, NextActions: []MachineAction{}}
}

func NormalizeMachine(r *CommandResult) {
	m := newMachine()
	warnSeen := map[string]bool{}
	constraintSeen := map[string]bool{}
	actionSeen := map[string]bool{}
	addWarn := func(code, msg string) {
		if code == "" || warnSeen[code] {
			return
		}
		warnSeen[code] = true
		m.Warnings = append(m.Warnings, MachineSignal{Code: code, Message: msg})
	}
	addConstraint := func(code, msg string) {
		if code == "" || constraintSeen[code] {
			return
		}
		constraintSeen[code] = true
		m.Constraints = append(m.Constraints, MachineSignal{Code: code, Blocking: true, Message: msg})
	}
	addAction := func(id, command, reason string) {
		if id == "" || actionSeen[id] {
			return
		}
		actionSeen[id] = true
		m.NextActions = append(m.NextActions, MachineAction{ID: id, Command: command, Issue: r.Issue, ReasonCode: reason})
	}
	for _, w := range r.Warnings {
		code := stableCode(w)
		if code == "" {
			code = "HUMAN_WARNING"
		}
		addWarn(code, w)
	}
	if r.Error != nil {
		addConstraint(r.Error.Code, r.Error.Message)
		switch r.Error.Code {
		case "MIGRATION_REQUIRED":
			addAction("migrate", "migrate", "MIGRATION_REQUIRED")
		case "CONTRACT_CHANGED":
			addAction("scope_reconcile", "scope", "CONTRACT_CHANGED")
		case "VALIDATION_FAILED":
			addAction("validate", "checkpoint", "VALIDATION_FAILED")
		case "REVIEW_REQUIRED":
			addAction("review", "review", "REVIEW_REQUIRED")
		case "STALE_GIT_STATE", "RESTORE_CONFLICT":
			addAction("inspect_git", "status", r.Error.Code)
		case "AUTH_REQUIRED":
			addAction("authenticate_gh", "doctor", "AUTH_REQUIRED")
		case "DEPENDENCY_MISSING":
			addAction("install_dependency", "doctor", "DEPENDENCY_MISSING")
		case "ADAPTER_INCOMPATIBLE":
			addAction("update_adapter", "adapter update", "ADAPTER_INCOMPATIBLE")
		case "ADAPTER_UNMANAGED":
			addAction("inspect_adapter_target", "adapter status", "ADAPTER_UNMANAGED")
		case "GITHUB_ISSUE_CLOSED":
			addAction("reopen_github_issue", "gh issue reopen", "GITHUB_ISSUE_CLOSED")
		}
	}
	for _, w := range r.Warnings {
		switch stableCode(w) {
		case "WORKFLOW_V1_COMPAT":
			addAction("migrate", "migrate", "WORKFLOW_V1_COMPAT")
		case "CONTRACT_CHANGED":
			addConstraint("CONTRACT_CHANGED", w)
			addAction("scope_reconcile", "scope", "CONTRACT_CHANGED")
		case "SCOPE_UNRESOLVED":
			addConstraint("SCOPE_UNRESOLVED", w)
		case "STALE_GIT_STATE":
			addConstraint("STALE_GIT_STATE", w)
		case "GITHUB_ISSUE_CLOSED":
			addConstraint("GITHUB_ISSUE_CLOSED", "GitHub Issue is closed; reopen it explicitly before active workflow mutation")
			addAction("reopen_github_issue", "gh issue reopen", "GITHUB_ISSUE_CLOSED")
		}
	}
	if strings.EqualFold(r.State, "BLOCKED") || strings.EqualFold(r.Lifecycle, "BLOCKED") {
		addConstraint("BLOCKED", "workflow is blocked")
	}
	for i, c := range r.Constraints {
		code := "WORKFLOW_CONSTRAINT"
		if i == 0 && r.Error != nil {
			code = r.Error.Code
		}
		addConstraint(code, c)
	}
	m.AllowedActions = allowedActions(r.Lifecycle, r.State)
	issueClosed := warnSeen["GITHUB_ISSUE_CLOSED"] || (r.Error != nil && r.Error.Code == "GITHUB_ISSUE_CLOSED")
	if issueClosed {
		m.AllowedActions = []string{"context", "status"}
		if warnSeen["WORKFLOW_V1_COMPAT"] || (r.Error != nil && r.Error.Code == "MIGRATION_REQUIRED") {
			m.AllowedActions = append(m.AllowedActions, "migrate")
		}
	}
	if len(m.NextActions) == 0 {
		switch canonicalLifecycle(r.Lifecycle, r.State) {
		case "NEEDS_ANALYSIS":
			addAction("analyze", "context", "NEEDS_ANALYSIS")
		case "READY":
			addAction("start", "start", "READY")
		case "IN_PROGRESS":
			addAction("continue_work", "checkpoint", "IN_PROGRESS")
		case "PAUSED":
			addAction("resume", "resume", "PAUSED")
		case "WAITING":
			addAction("recheck", "recheck", "WAITING")
		case "DEFERRED":
			addAction("resume", "resume", "DEFERRED")
		case "BLOCKED":
			addAction("resolve_blocker", "context", "BLOCKED")
		case "COMPLETED", "CANCELLED":
			addAction("reopen", "reopen", canonicalLifecycle(r.Lifecycle, r.State))
		}
	}
	r.Machine = m
}

func stableCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return ""
		}
	}
	return s
}
func canonicalLifecycle(lifecycle, state string) string {
	if lifecycle != "" {
		return strings.ToUpper(lifecycle)
	}
	switch strings.ToUpper(state) {
	case "DONE":
		return "COMPLETED"
	case "REVIEW", "FIX":
		return "IN_PROGRESS"
	default:
		return strings.ToUpper(state)
	}
}
func allowedActions(lifecycle, state string) []string {
	switch canonicalLifecycle(lifecycle, state) {
	case "NEEDS_ANALYSIS":
		return []string{"context", "status", "cancel"}
	case "READY":
		return []string{"start", "cancel"}
	case "IN_PROGRESS":
		return []string{"resume", "checkpoint", "handoff", "pause", "wait", "defer", "block", "scope", "review", "fix", "complete", "cancel"}
	case "PAUSED":
		return []string{"resume", "block", "cancel"}
	case "WAITING":
		return []string{"recheck", "block", "cancel"}
	case "DEFERRED":
		return []string{"resume", "cancel"}
	case "BLOCKED":
		return []string{"context", "resume", "scope", "cancel"}
	case "COMPLETED", "CANCELLED":
		return []string{"reopen"}
	default:
		return []string{}
	}
}

const (
	ExitOK          = 0
	ExitInvalid     = 2
	ExitEnvironment = 3
	ExitGitHub      = 4
	ExitWorkflow    = 5
	ExitValidation  = 6
	ExitConflict    = 7
	ExitProtocol    = 8
	ExitInternal    = 9
)

func ExitCodeFor(c string) int {
	switch c {
	case "INVALID_INVOCATION", "ADAPTER_NOT_FOUND", "ADAPTER_NOT_READY":
		return ExitInvalid
	case "AUTH_REQUIRED", "GITHUB_PERMISSION_DENIED", "REPOSITORY_NOT_FOUND", "ISSUE_NOT_FOUND", "GITHUB_UNAVAILABLE":
		return ExitGitHub
	case "TRANSITION_BLOCKED", "BLOCKED", "SCOPE_UNRESOLVED", "MIGRATION_REQUIRED", "WAITING", "DEFERRED", "GITHUB_ISSUE_CLOSED":
		return ExitWorkflow
	case "VALIDATION_FAILED", "REVIEW_REQUIRED":
		return ExitValidation
	case "STALE_GIT_STATE", "CONFLICT", "EVENT_FORK", "CONTRACT_CHANGED", "RESTORE_CONFLICT", "ADAPTER_UNMANAGED":
		return ExitConflict
	case "PROTOCOL_ERROR", "UNSUPPORTED_SCHEMA", "EVENT_CHAIN_BROKEN", "TASK_CONTRACT_INVALID", "PROJECT_CONFIG_INVALID", "ADAPTER_MANIFEST_INVALID", "ADAPTER_INCOMPATIBLE":
		return ExitProtocol
	case "DEPENDENCY_MISSING", "ENVIRONMENT_INVALID", "NOT_A_GIT_REPOSITORY", "REMOTE_NOT_FOUND", "REPOSITORY_IDENTITY_MISMATCH", "COMMAND_TIMEOUT", "ADAPTER_INSTALL_FAILED":
		return ExitEnvironment
	default:
		return ExitInternal
	}
}
