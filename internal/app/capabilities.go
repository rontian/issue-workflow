package app

import (
	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

const CapabilitiesSchema = "iw.capabilities/v1"

type CapabilityCommand struct {
	Name     string `json:"name"`
	Mutation bool   `json:"mutation"`
	DryRun   bool   `json:"dry_run"`
}

type Capabilities struct {
	Schema              string                  `json:"schema"`
	Protocol            string                  `json:"protocol"`
	CommandResultSchema string                  `json:"command_result_schema"`
	ContextSchema       string                  `json:"context_schema"`
	ProjectSchema       string                  `json:"project_schema"`
	AdapterSchema       string                  `json:"adapter_schema"`
	WorkflowSchemas     []string                `json:"workflow_schemas"`
	TaskContractSchemas []string                `json:"task_contract_schemas"`
	Lifecycles          []domain.Lifecycle      `json:"lifecycles"`
	ValidationStatuses  []string                `json:"validation_statuses"`
	Commands            []CapabilityCommand     `json:"commands"`
	Compatibility       map[string][]string     `json:"compatibility"`
}

func CapabilitiesResult() result.CommandResult {
	commands := []CapabilityCommand{
		{Name: "version"}, {Name: "capabilities"}, {Name: "doctor"}, {Name: "init"},
		{Name: "project status"}, {Name: "tasks"}, {Name: "next"}, {Name: "context"}, {Name: "status"},
		{Name: "new", Mutation: true, DryRun: true}, {Name: "migrate", Mutation: true, DryRun: true},
		{Name: "start", Mutation: true, DryRun: true}, {Name: "resume", Mutation: true, DryRun: true},
		{Name: "pause", Mutation: true, DryRun: true}, {Name: "wait", Mutation: true, DryRun: true},
		{Name: "recheck", Mutation: true, DryRun: true}, {Name: "defer", Mutation: true, DryRun: true},
		{Name: "restore", Mutation: true, DryRun: true}, {Name: "checkpoint", Mutation: true, DryRun: true},
		{Name: "handoff", Mutation: true, DryRun: true}, {Name: "block", Mutation: true, DryRun: true},
		{Name: "scope propose", Mutation: true, DryRun: true}, {Name: "scope accept", Mutation: true, DryRun: true}, {Name: "scope reject", Mutation: true, DryRun: true},
		{Name: "review", Mutation: true, DryRun: true}, {Name: "fix", Mutation: true, DryRun: true},
		{Name: "complete", Mutation: true, DryRun: true}, {Name: "final", Mutation: true, DryRun: true},
		{Name: "cancel", Mutation: true, DryRun: true}, {Name: "reopen", Mutation: true, DryRun: true},
		{Name: "adapter list"}, {Name: "adapter status"}, {Name: "adapter doctor"},
		{Name: "adapter install", Mutation: true, DryRun: true}, {Name: "adapter update", Mutation: true, DryRun: true}, {Name: "adapter remove", Mutation: true, DryRun: true},
	}
	data := Capabilities{
		Schema: CapabilitiesSchema, Protocol: "2.0", CommandResultSchema: result.Schema,
		ContextSchema: "iw.context/v1", ProjectSchema: "iw.project/v1", AdapterSchema: "iw.adapter/v1",
		WorkflowSchemas: []string{"iw.workflow-event/v1", "iw.workflow-event/v2"}, TaskContractSchemas: []string{"iw.task-contract/v1"},
		Lifecycles: append([]domain.Lifecycle(nil), domain.LifecycleValues...),
		ValidationStatuses: []string{"pass", "fail", "not_run", "skipped", "manual_required", "not_applicable"},
		Commands: commands,
		Compatibility: map[string][]string{
			"read": {"iw.workflow-event/v1", "iw.task-contract/v1"},
			"aliases": {"final -> complete"},
		},
	}
	return result.Success("capabilities", data)
}
