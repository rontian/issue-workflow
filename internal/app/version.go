package app

import (
	"github.com/rontian/issue-workflow/internal/result"
	"github.com/rontian/issue-workflow/internal/version"
)

func VersionResult() result.CommandResult {
	return result.Success("version", version.Info{Version: version.Version, Protocol: version.Protocol, Commit: version.Commit, BuildDate: version.BuildDate})
}
