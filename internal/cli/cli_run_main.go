package cli

import (
	"context"
	"os"

	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/result"
)

func (c *CLI) Run(ctx context.Context, args []string) int {
	if c.Stdout == nil { c.Stdout = os.Stdout }
	if c.Stderr == nil { c.Stderr = os.Stderr }
	if c.Getwd == nil { c.Getwd = os.Getwd }
	if c.IsTerminal == nil { c.IsTerminal = func() bool { return false } }
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		c.printUsage(c.Stdout)
		return result.ExitOK
	}
	command, o, help, err := parse(args)
	if help { c.printCommandUsage(command, c.Stdout); return result.ExitOK }
	if err != nil { return c.render(result.Failure(commandOrUnknown(command), "INVALID_INVOCATION", err.Error(), nil, map[string]any{}), result.ExitInvalid, containsJSONFlag(args)) }
	cwd, err := c.Getwd()
	if err != nil { return c.render(result.Failure(command, "ENVIRONMENT_INVALID", "无法读取当前目录", nil, nil), result.ExitEnvironment, o.JSON) }
	if command != "version" && c.Workflow == nil && command != "doctor" && command != "init" { return c.render(result.Failure(command, "ENVIRONMENT_INVALID", "workflow service unavailable", nil, nil), result.ExitEnvironment, o.JSON) }
	wo := app.WorkflowOptions{CWD: cwd, Repo: o.Repo, Remote: o.Remote, Host: o.Host, Issue: o.Issue}
	if handled, exit := c.runEnvironmentOrRead(ctx, command, o, wo, cwd); handled { return exit }
	return c.runMutation(ctx, command, o, wo)
}
