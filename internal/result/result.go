package result

const Schema = "iw.command-result/v1"

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
type CommandResult struct {
	Schema      string   `json:"schema"`
	OK          bool     `json:"ok"`
	Command     string   `json:"command"`
	Repository  string   `json:"repository,omitempty"`
	Issue       *int     `json:"issue,omitempty"`
	State       string   `json:"state,omitempty"`
	Warnings    []string `json:"warnings"`
	Constraints []string `json:"constraints"`
	NextActions []string `json:"next_actions"`
	Data        any      `json:"data"`
	Error       *Error   `json:"error"`
}

func Success(c string, d any) CommandResult {
	return CommandResult{Schema: Schema, OK: true, Command: c, Warnings: []string{}, Constraints: []string{}, NextActions: []string{}, Data: d}
}
func Failure(c, code, msg string, details map[string]any, d any) CommandResult {
	return CommandResult{Schema: Schema, OK: false, Command: c, Warnings: []string{}, Constraints: []string{}, NextActions: []string{}, Data: d, Error: &Error{Code: code, Message: msg, Details: details}}
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
	case "INVALID_INVOCATION":
		return ExitInvalid
	case "AUTH_REQUIRED", "GITHUB_PERMISSION_DENIED", "REPOSITORY_NOT_FOUND", "ISSUE_NOT_FOUND", "GITHUB_UNAVAILABLE":
		return ExitGitHub
	case "TRANSITION_BLOCKED", "BLOCKED", "SCOPE_UNRESOLVED":
		return ExitWorkflow
	case "VALIDATION_FAILED":
		return ExitValidation
	case "STALE_GIT_STATE", "CONFLICT", "EVENT_FORK", "CONTRACT_CHANGED":
		return ExitConflict
	case "PROTOCOL_ERROR", "UNSUPPORTED_SCHEMA", "EVENT_CHAIN_BROKEN", "TASK_CONTRACT_INVALID":
		return ExitProtocol
	case "DEPENDENCY_MISSING", "ENVIRONMENT_INVALID", "NOT_A_GIT_REPOSITORY", "REMOTE_NOT_FOUND", "REPOSITORY_IDENTITY_MISMATCH", "COMMAND_TIMEOUT":
		return ExitEnvironment
	default:
		return ExitInternal
	}
}
