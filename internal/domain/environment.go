package domain

type RepositoryIdentity struct {
	Host  string `json:"host"`
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func (r RepositoryIdentity) FullName() string {
	if r.Owner == "" || r.Repo == "" {
		return ""
	}
	return r.Owner + "/" + r.Repo
}

type GitSnapshot struct {
	RepositoryRoot string   `json:"repository_root"`
	Branch         string   `json:"branch,omitempty"`
	Head           string   `json:"head"`
	Detached       bool     `json:"detached"`
	Dirty          bool     `json:"dirty"`
	ChangedPaths   []string `json:"changed_paths"`
}

type GitHubRepository struct {
	NameWithOwner    string `json:"name_with_owner"`
	URL              string `json:"url,omitempty"`
	HasIssuesEnabled bool   `json:"has_issues_enabled"`
}

type GitHubIssue struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
}
