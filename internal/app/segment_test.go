package app

import(
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
)

func TestProjectReviewPolicyBindsReviewToCurrentHead(t *testing.T){
	s,_,git:=stateService(t,false);root:=t.TempDir();git.snap.RepositoryRoot=root;if err:=os.WriteFile(filepath.Join(root,"iw.json"),[]byte(`{"schema":"iw.project/v1","policy":{"require_review":true}}`),0644);err!=nil{t.Fatal(err)}
	if r,c:=s.Start(context.Background(),mo("p-start"));c!=0||!r.OK{t.Fatal(c,r)}
	rv:=mo("p-review-a");rv.Summary="clean abc";if r,c:=s.Review(context.Background(),rv);c!=0||!r.OK{t.Fatal(c,r)}
	git.snap.Head="def";cp:=mo("p-cp-def");cp.Summary="changed after review";cp.Next="review";if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatal(c,r)}
	fn:=mo("p-complete");fn.Summary="done";if r,c:=s.Complete(context.Background(),fn);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("stale review %d %#v",c,r)}
	rv2:=mo("p-review-def");rv2.Summary="clean def";if r,c:=s.Review(context.Background(),rv2);c!=0||!r.OK{t.Fatal(c,r)}
	if r,c:=s.Complete(context.Background(),fn);c!=0||!r.OK||r.State!="COMPLETED"{t.Fatalf("complete %d %#v",c,r)}
}

func TestReopenStartsNewEvidenceSegment(t *testing.T){
	s,_,git:=stateService(t,true);git.snap.RepositoryRoot=t.TempDir();if r,c:=s.Start(context.Background(),mo("seg-start1"));c!=0||!r.OK{t.Fatal(c,r)}
	cp:=mo("seg-cp1");cp.Summary="done";cp.Next="complete";cp.Validation=[]domain.ValidationRecord{{ID:"v1",Status:"pass"}};if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatal(c,r)}
	fn:=mo("seg-complete1");fn.Summary="done";if r,c:=s.Complete(context.Background(),fn);c!=0||!r.OK{t.Fatalf("complete1 %d %#v",c,r)}
	rp:=mo("seg-reopen");rp.Reason="new segment";if r,c:=s.Reopen(context.Background(),rp);c!=0||!r.OK||r.State!="READY"{t.Fatalf("reopen %d %#v",c,r)}
	if r,c:=s.Start(context.Background(),mo("seg-start2"));c!=0||!r.OK{t.Fatalf("start2 %d %#v",c,r)}
	fn2:=mo("seg-complete2");fn2.Summary="second done";if r,c:=s.Complete(context.Background(),fn2);c!=6||r.Error==nil||r.Error.Code!="VALIDATION_FAILED"{t.Fatalf("old evidence leaked %d %#v",c,r)}
}
