package app

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/rontian/issue-workflow/internal/domain"
)

func TestDeferRequiresCleanPublishedCurrentHead(t *testing.T) {
    tests:=[]struct{
        name string
        mutate func(*mutableGit)
    }{
        {"dirty worktree",func(g *mutableGit){g.snap.Dirty=true;g.snap.ChangedPaths=[]string{"dirty.go"}}},
        {"Git HEAD drift",func(g *mutableGit){g.snap.Head="def";g.remoteHead="def"}},
        {"unpublished HEAD",func(g *mutableGit){g.remoteHead="old"}},
        {"detached",func(g *mutableGit){g.snap.Detached=true;g.snap.Branch=""}},
    }
    for _,tt:=range tests{
        t.Run(tt.name,func(t *testing.T){
            s,gh,git:=stateService(t,false)
            if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("start: %d %#v",c,r)}
            tt.mutate(git)
            o:=mo("defer");o.Reason="later milestone";o.Next="resume after milestone"
            r,c:=s.Defer(context.Background(),o)
            if c!=7||r.Error==nil||r.Error.Code!="STALE_GIT_STATE"{t.Fatalf("defer: %d %#v",c,r)}
            if gh.appendCalls!=1{t.Fatalf("rejected defer must not append; calls=%d",gh.appendCalls)}
        })
    }
}

func TestDeferRequiresResolvedReviewAtCurrentHead(t *testing.T){
    s,gh,git:=stateService(t,false)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    rv:=mo("review");rv.Findings=[]string{"F01"}
    if r,c:=s.Review(context.Background(),rv);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    d:=mo("defer");d.Reason="later milestone";d.Next="resume after dependency"
    if r,c:=s.Defer(context.Background(),d);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("unresolved review: %d %#v",c,r)}
    fx:=mo("fix");fx.Summary="fixed F01"
    if r,c:=s.Fix(context.Background(),fx);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    rv2:=mo("review2")
    if r,c:=s.Review(context.Background(),rv2);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    git.snap.Head="def"
    cp:=mo("cp");cp.Summary="changed after review";cp.Next="review again"
    if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    git.remoteHead="def"
    if r,c:=s.Defer(context.Background(),d);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("stale review: %d %#v",c,r)}
    if gh.appendCalls!=5{t.Fatalf("unexpected appended event count=%d",gh.appendCalls)}
}

func TestDeferHonorsProjectReviewPolicy(t *testing.T){
    s,_,git:=stateService(t,false)
    root:=t.TempDir()
    git.snap.RepositoryRoot=root
    err:=os.WriteFile(filepath.Join(root,"iw.json"),[]byte(`{"schema":"iw.project/v1","policy":{"require_review":true}}`),0644)
    if err!=nil{t.Fatal(err)}
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    d:=mo("defer");d.Reason="future acceptance";d.Next="resume after acceptance starts"
    if r,c:=s.Defer(context.Background(),d);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("missing required review: %d %#v",c,r)}
    rv:=mo("review")
    if r,c:=s.Review(context.Background(),rv);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    if r,c:=s.Defer(context.Background(),d);c!=0||!r.OK||r.State!="DEFERRED"{t.Fatalf("ready defer: %d %#v",c,r)}
}

func TestPassedReviewAtOldHeadCannotCompleteEvenWithoutReviewPolicy(t *testing.T){
    s,_,git:=stateService(t,false)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    if r,c:=s.Review(context.Background(),mo("review"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    git.snap.Head="def"
    cp:=mo("checkpoint");cp.Summary="new code";cp.Next="review again";cp.Validation=[]domain.ValidationRecord{}
    if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    done:=mo("complete");done.Summary="all done"
    if r,c:=s.Complete(context.Background(),done);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("stale review: %d %#v",c,r)}
}
