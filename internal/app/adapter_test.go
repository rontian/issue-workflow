package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rontian/issue-workflow/internal/result"
)

type localAdapterSource struct{ dir string }
func (s localAdapterSource) Checkout(context.Context,string)(string,func(),error){return s.dir,func(){},nil}

func adapterFixture(t *testing.T, compatible bool)(string,map[string]AdapterRegistryEntry){
	t.Helper();src:=t.TempDir();asset:=filepath.Join(src,"skill");if err:=os.MkdirAll(asset,0755);err!=nil{t.Fatal(err)};if err:=os.WriteFile(filepath.Join(asset,"SKILL.md"),[]byte("v1\n"),0644);err!=nil{t.Fatal(err)}
	m:=AdapterManifest{Schema:AdapterSchema,ID:"test",Name:"Test Adapter",Host:"test"};m.Source.Repository="rontian/test-adapter";m.Requires.TaskContractSchemas=[]string{"iw.task-contract/v1"};m.Requires.WorkflowEventSchemas=[]string{"iw.workflow-event/v2"};if compatible{m.Requires.CommandResultSchemas=[]string{result.Schema}}else{m.Requires.CommandResultSchemas=[]string{result.LegacySchema}};m.Assets=[]AdapterAsset{{Kind:"test-skill",Source:"skill",Name:"workflow"}};raw,_:=json.Marshal(m);if err:=os.WriteFile(filepath.Join(src,"iw-adapter.json"),raw,0644);err!=nil{t.Fatal(err)}
	reg:=map[string]AdapterRegistryEntry{"test":{ID:"test",Repository:"rontian/test-adapter",Status:"available",RootEnv:"TEST_ADAPTER_HOME",AssetTargets:map[string]string{"test-skill":"skills/{name}"}}};return src,reg
}

func TestAdapterInstallUpdateRemoveLifecycle(t *testing.T){
	src,reg:=adapterFixture(t,true);root:=t.TempDir();s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(k string)string{if k=="TEST_ADAPTER_HOME"{return root};return ""}}
	target:=filepath.Join(root,"skills","workflow")
	if r,c:=s.Install(context.Background(),"test",true);c!=0||!r.OK{t.Fatalf("dry install %d %#v",c,r)};if _,err:=os.Stat(target);!os.IsNotExist(err){t.Fatalf("dry-run created target: %v",err)}
	if r,c:=s.Install(context.Background(),"test",false);c!=0||!r.OK{t.Fatalf("install %d %#v",c,r)};if _,err:=os.Stat(filepath.Join(target,"SKILL.md"));err!=nil{t.Fatal(err)};if m,ok:=readManagedMarker(target);!ok||m.AdapterID!="test"{t.Fatalf("marker %#v %v",m,ok)}
	if r,c:=s.Status(context.Background(),"test");c!=0||!r.OK||!r.Data.(AdapterStatus).Installed||!r.Data.(AdapterStatus).Managed{t.Fatalf("status %d %#v",c,r)}
	if err:=os.WriteFile(filepath.Join(src,"skill","SKILL.md"),[]byte("v2\n"),0644);err!=nil{t.Fatal(err)};if r,c:=s.Update(context.Background(),"test",false);c!=0||!r.OK{t.Fatalf("update %d %#v",c,r)};got,_:=os.ReadFile(filepath.Join(target,"SKILL.md"));if string(got)!="v2\n"{t.Fatalf("got %q",got)}
	if r,c:=s.Remove(context.Background(),"test",true);c!=0||!r.OK{t.Fatalf("dry remove %d %#v",c,r)};if _,err:=os.Stat(target);err!=nil{t.Fatal(err)}
	if r,c:=s.Remove(context.Background(),"test",false);c!=0||!r.OK{t.Fatalf("remove %d %#v",c,r)};if _,err:=os.Stat(target);!os.IsNotExist(err){t.Fatalf("target remains: %v",err)}
}

func TestAdapterProtectsUnmanagedTarget(t *testing.T){
	src,reg:=adapterFixture(t,true);root:=t.TempDir();target:=filepath.Join(root,"skills","workflow");if err:=os.MkdirAll(target,0755);err!=nil{t.Fatal(err)};if err:=os.WriteFile(filepath.Join(target,"user.txt"),[]byte("keep"),0644);err!=nil{t.Fatal(err)}
	s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(string)string{return root}};r,c:=s.Install(context.Background(),"test",false);if c!=result.ExitConflict||r.Error==nil||r.Error.Code!="ADAPTER_UNMANAGED"{t.Fatalf("%d %#v",c,r)};if _,err:=os.Stat(filepath.Join(target,"user.txt"));err!=nil{t.Fatal("user data changed")}
}

func TestAdapterRejectsIncompatibleManifest(t *testing.T){
	src,reg:=adapterFixture(t,false);root:=t.TempDir();s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(string)string{return root}};r,c:=s.Install(context.Background(),"test",false);if c!=result.ExitProtocol||r.Error==nil||r.Error.Code!="ADAPTER_INCOMPATIBLE"{t.Fatalf("%d %#v",c,r)}
}

func TestAdapterManifestRejectsTraversal(t *testing.T){
	src,reg:=adapterFixture(t,true);raw,err:=os.ReadFile(filepath.Join(src,"iw-adapter.json"));if err!=nil{t.Fatal(err)};var m AdapterManifest;if json.Unmarshal(raw,&m)!=nil{t.Fatal("decode")};m.Assets[0].Source="../secret";raw,_=json.Marshal(m);if os.WriteFile(filepath.Join(src,"iw-adapter.json"),raw,0644)!=nil{t.Fatal("write")};s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(string)string{return t.TempDir()}};r,c:=s.Install(context.Background(),"test",false);if c!=result.ExitProtocol||r.Error==nil||r.Error.Code!="ADAPTER_MANIFEST_INVALID"{t.Fatalf("%d %#v",c,r)}
}


func TestAdapterUpdateDoesNotDeleteUnrelatedBackupSibling(t *testing.T) {
	src,reg:=adapterFixture(t,true)
	root:=t.TempDir()
	s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(string)string{return root}}
	if r,c:=s.Install(context.Background(),"test",false);c!=0||!r.OK{
		t.Fatalf("initial install: %d %#v",c,r)
	}
	target:=filepath.Join(root,"skills","workflow")
	// This directory is NOT a managed adapter asset, even though its name
	// matches the old implementation's fixed backup path.
	unrelated:=target+".iw-backup"
	if err:=os.MkdirAll(unrelated,0755);err!=nil{t.Fatal(err)}
	sentinel:=filepath.Join(unrelated,"user-owned.txt")
	if err:=os.WriteFile(sentinel,[]byte("must be preserved"),0644);err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(filepath.Join(src,"skill","SKILL.md"),[]byte("updated\n"),0644);err!=nil{t.Fatal(err)}
	if r,c:=s.Update(context.Background(),"test",false);c!=0||!r.OK{
		t.Fatalf("adapter update: %d %#v",c,r)
	}
	if got,err:=os.ReadFile(sentinel);err!=nil||string(got)!="must be preserved"{
		t.Fatalf("unrelated user-owned sibling was changed: %q %v",got,err)
	}
	if got,err:=os.ReadFile(filepath.Join(target,"SKILL.md"));err!=nil||string(got)!="updated\n"{
		t.Fatalf("updated managed content missing: %q %v",got,err)
	}
}


func TestAdapterRejectsLocallyModifiedManagedTree(t *testing.T) {
	for _,kind:=range []string{"added","edited","deleted","missing-inventory"} {
		t.Run(kind,func(t *testing.T){
			src,reg:=adapterFixture(t,true)
			root:=t.TempDir()
			s:=&AdapterService{Source:localAdapterSource{src},Registry:reg,Getenv:func(string)string{return root}}
			if r,c:=s.Install(context.Background(),"test",false);c!=0||!r.OK{t.Fatalf("install: %d %#v",c,r)}
			target:=filepath.Join(root,"skills","workflow")
			skill:=filepath.Join(target,"SKILL.md")
			switch kind {
			case "added":
				if err:=os.WriteFile(filepath.Join(target,"user-note.txt"),[]byte("local data"),0644);err!=nil{t.Fatal(err)}
			case "edited":
				if err:=os.WriteFile(skill,[]byte("local edits"),0644);err!=nil{t.Fatal(err)}
			case "deleted":
				if err:=os.Remove(skill);err!=nil{t.Fatal(err)}
			case "missing-inventory":
				marker,ok:=readManagedMarker(target);if !ok{t.Fatal("marker missing")}
				marker.ManagedFiles=nil
				raw,err:=json.Marshal(marker);if err!=nil{t.Fatal(err)}
				if err:=os.WriteFile(filepath.Join(target,".iw-adapter-install.json"),raw,0644);err!=nil{t.Fatal(err)}
			}
			for _,operation:=range []struct {
				name string
				run func(context.Context,string,bool)(result.CommandResult,int)
			}{
				{"update",s.Update},
				{"remove",s.Remove},
			}{
				r,c:=operation.run(context.Background(),"test",false)
				if c!=result.ExitConflict||r.Error==nil||r.Error.Code!="ADAPTER_UNMANAGED"{
					t.Fatalf("%s must refuse locally modified/legacy target: %d %#v",operation.name,c,r)
				}
			}
			if _,err:=os.Stat(target);err!=nil{t.Fatalf("refused operation removed target: %v",err)}
			if kind=="added" {
				if got,err:=os.ReadFile(filepath.Join(target,"user-note.txt"));err!=nil||string(got)!="local data"{
					t.Fatalf("locally added content changed: %q %v",got,err)
				}
			}
		})
	}
}
