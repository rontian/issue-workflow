package ghcli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
)

func (c *Client) TaskIssues(ctx context.Context, repo domain.RepositoryIdentity) ([]domain.GitHubIssueDetails, error) {
	endpoint:=fmt.Sprintf("repos/%s/%s/issues?state=all&per_page=100",repo.Owner,repo.Repo)
	out,err:=c.runWithEnv(ctx,repoEnv(repo),"api","--paginate","--slurp",endpoint,"-H","Accept: application/vnd.github+json");if err!=nil{return nil,err}
	var pages [][]struct{ Number int `json:"number"`; State string `json:"state"`; Title string `json:"title"`; Body string `json:"body"`; HTMLURL string `json:"html_url"`; PullRequest any `json:"pull_request"` }
	if err:=json.Unmarshal(out,&pages);err!=nil{return nil,fmt.Errorf("decode task issues: %w",err)}
	res:=[]domain.GitHubIssueDetails{}
	for _,page:=range pages{for _,x:=range page{if x.PullRequest!=nil{continue};if !strings.Contains(x.Body,"<!-- iw:task-contract:"){continue};res=append(res,domain.GitHubIssueDetails{Number:x.Number,State:strings.ToUpper(x.State),Title:x.Title,Body:x.Body,URL:x.HTMLURL})}}
	return res,nil
}
