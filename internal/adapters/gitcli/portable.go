package gitcli

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) RemoteBranchHead(ctx context.Context, dir, remote, branch string) (string, error) {
	if strings.TrimSpace(remote)=="" || strings.TrimSpace(branch)=="" { return "", fmt.Errorf("remote and branch are required") }
	out, err := c.run(ctx, dir, "ls-remote", "--heads", remote, "refs/heads/"+branch)
	if err != nil { return "", err }
	fields := strings.Fields(out); if len(fields)<2 { return "", fmt.Errorf("remote branch not found: %s/%s", remote, branch) }
	return fields[0], nil
}

func (c *Client) FetchRemote(ctx context.Context, dir, remote string) error {
	_, err := c.run(ctx, dir, "fetch", "--prune", remote); return err
}

func (c *Client) CommitExists(ctx context.Context, dir, head string) bool {
	_, err := c.run(ctx, dir, "cat-file", "-e", head+"^{commit}"); return err==nil
}

func (c *Client) RestoreBranch(ctx context.Context, dir, remote, branch, head string) error {
	remoteHead, err := c.RemoteBranchHead(ctx,dir,remote,branch); if err!=nil { return err }
	if remoteHead!=head { return fmt.Errorf("remote branch head %s does not match portable handoff %s",remoteHead,head) }
	localHead, localErr := c.run(ctx,dir,"show-ref","--verify","--hash","refs/heads/"+branch)
	if localErr==nil {
		if _,err:=c.run(ctx,dir,"switch",branch);err!=nil{return err}
		if localHead!=head { if _,err:=c.run(ctx,dir,"merge","--ff-only",head);err!=nil{return fmt.Errorf("local branch cannot be safely fast-forwarded to portable handoff: %w",err)} }
	} else {
		if _,err:=c.run(ctx,dir,"switch","-c",branch,"--track",remote+"/"+branch);err!=nil{return err}
	}
	got,err:=c.run(ctx,dir,"rev-parse","--verify","HEAD");if err!=nil{return err};if got!=head{return fmt.Errorf("restored HEAD %s does not match expected %s",got,head)}
	return nil
}
