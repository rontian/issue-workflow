package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

const (
	AdapterSchema = "iw.adapter/v1"
	AdapterStatusSchema = "iw.adapter-status/v1"
	AdapterInstallMarkerSchema = "iw.adapter-install/v1"
)

type AdapterManifest struct {
	Schema string `json:"schema"`
	ID string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
	Source struct { Repository string `json:"repository"` } `json:"source"`
	Requires struct {
		CommandResultSchemas []string `json:"command_result_schemas"`
		TaskContractSchemas []string `json:"task_contract_schemas"`
		WorkflowEventSchemas []string `json:"workflow_event_schemas"`
	} `json:"requires"`
	Assets []AdapterAsset `json:"assets"`
}

type AdapterAsset struct {
	Kind string `json:"kind"`
	Source string `json:"source"`
	Name string `json:"name"`
}

type AdapterRegistryEntry struct {
	ID string `json:"id"`
	Repository string `json:"repository"`
	Status string `json:"status"`
	RootEnv string `json:"root_env,omitempty"`
	DefaultRootRelative string `json:"default_root_relative,omitempty"`
	AssetTargets map[string]string `json:"asset_targets,omitempty"`
}

var OfficialAdapterRegistry = map[string]AdapterRegistryEntry{
	"codex": {ID:"codex", Repository:"rontian/codex-workflow", Status:"available", RootEnv:"CODEX_HOME", DefaultRootRelative:".codex", AssetTargets:map[string]string{"codex-skill":"skills/{name}"}},
	"pi": {ID:"pi", Repository:"rontian/pi-workflow", Status:"planned"},
}

type AdapterSource interface { Checkout(context.Context,string)(string,func(),error) }

type GitHubAdapterSource struct { Runner ports.Runner }
func (s GitHubAdapterSource) Checkout(ctx context.Context, repo string)(string,func(),error){
	if s.Runner==nil{return "",func(){},fmt.Errorf("runner unavailable")}
	if _,err:=s.Runner.LookPath("gh");err!=nil{return "",func(){},fmt.Errorf("gh command not found")}
	tmp,err:=os.MkdirTemp("","iw-adapter-*");if err!=nil{return "",func(){},err};cleanup:=func(){_ = os.RemoveAll(tmp)};dest:=filepath.Join(tmp,"source")
	_,err=s.Runner.Run(ctx,ports.Command{Name:"gh",Args:[]string{"repo","clone",repo,dest,"--","--depth","1"}});if err!=nil{cleanup();return "",func(){},fmt.Errorf("clone adapter source: %w",err)}
	return dest,cleanup,nil
}

type AdapterService struct {
	Runner ports.Runner
	Source AdapterSource
	Registry map[string]AdapterRegistryEntry
	Getenv func(string)string
	HomeDir func()(string,error)
}

type AdapterStatus struct {
	Schema string `json:"schema"`
	ID string `json:"id"`
	Repository string `json:"repository"`
	RegistryStatus string `json:"registry_status"`
	Compatible bool `json:"compatible"`
	Installed bool `json:"installed"`
	Managed bool `json:"managed"`
	Targets []string `json:"targets"`
	Manifest *AdapterManifest `json:"manifest,omitempty"`
}

type AdapterInstallMarker struct {
	Schema string `json:"schema"`
	AdapterID string `json:"adapter_id"`
	Repository string `json:"repository"`
	AssetKind string `json:"asset_kind"`
	AssetSource string `json:"asset_source"`
}

type AdapterActionReport struct { Status AdapterStatus `json:"status"`; DryRun bool `json:"dry_run"`; Actions []string `json:"actions"` }

func (s *AdapterService) registry()map[string]AdapterRegistryEntry{if s.Registry!=nil{return s.Registry};return OfficialAdapterRegistry}
func (s *AdapterService) source()AdapterSource{if s.Source!=nil{return s.Source};return GitHubAdapterSource{Runner:s.Runner}}
func (s *AdapterService) getenv(k string)string{if s.Getenv!=nil{return s.Getenv(k)};return os.Getenv(k)}
func (s *AdapterService) home()(string,error){if s.HomeDir!=nil{return s.HomeDir()};return os.UserHomeDir()}

func (s *AdapterService) List() (result.CommandResult,int) {
	ids:=[]string{"codex","pi"}; if s.Registry!=nil{ids=ids[:0];for id:=range s.Registry{ids=append(ids,id)};sortStrings(ids)}
	entries:=make([]AdapterRegistryEntry,0,len(ids));for _,id:=range ids{if e,ok:=s.registry()[id];ok{entries=append(entries,e)}}
	return result.Success("adapter list",map[string]any{"schema":"iw.adapter-registry/v1","adapters":entries}),result.ExitOK
}

func (s *AdapterService) load(ctx context.Context,id string)(AdapterRegistryEntry,AdapterManifest,string,func(),error){
	e,ok:=s.registry()[id];if !ok{return e,AdapterManifest{},"",func(){},fmt.Errorf("ADAPTER_NOT_FOUND: unknown adapter %q",id)}
	if e.Status!="available"{return e,AdapterManifest{},"",func(){},fmt.Errorf("ADAPTER_NOT_READY: adapter %q is %s",id,e.Status)}
	dir,cleanup,err:=s.source().Checkout(ctx,e.Repository);if err!=nil{return e,AdapterManifest{},"",func(){},err}
	raw,err:=os.ReadFile(filepath.Join(dir,"iw-adapter.json"));if err!=nil{cleanup();return e,AdapterManifest{},"",func(){},fmt.Errorf("ADAPTER_MANIFEST_INVALID: %w",err)}
	var m AdapterManifest;if err:=json.Unmarshal(raw,&m);err!=nil{cleanup();return e,m,"",func(){},fmt.Errorf("ADAPTER_MANIFEST_INVALID: %w",err)}
	if err:=validateAdapterManifest(e,m);err!=nil{cleanup();return e,m,"",func(){},err}
	return e,m,dir,cleanup,nil
}

func validateAdapterManifest(e AdapterRegistryEntry,m AdapterManifest)error{
	if m.Schema!=AdapterSchema||m.ID==""||m.ID!=e.ID||m.Source.Repository!=e.Repository{return fmt.Errorf("ADAPTER_MANIFEST_INVALID: manifest identity does not match registry")}
	if len(m.Assets)==0{return fmt.Errorf("ADAPTER_MANIFEST_INVALID: manifest has no assets")}
	for _,a:=range m.Assets{if a.Kind==""||a.Source==""||a.Name==""{return fmt.Errorf("ADAPTER_MANIFEST_INVALID: asset kind/source/name required")};if unsafeRel(a.Source){return fmt.Errorf("ADAPTER_MANIFEST_INVALID: unsafe asset source %q",a.Source)};if _,ok:=e.AssetTargets[a.Kind];!ok{return fmt.Errorf("ADAPTER_MANIFEST_INVALID: registry has no target for asset kind %q",a.Kind)}}
	return nil
}

func adapterCompatible(m AdapterManifest)bool{return containsString(m.Requires.CommandResultSchemas,result.Schema)&&containsString(m.Requires.TaskContractSchemas,"iw.task-contract/v1")&&containsString(m.Requires.WorkflowEventSchemas,"iw.workflow-event/v2")}

func (s *AdapterService) installRoot(e AdapterRegistryEntry)(string,error){
	root:=strings.TrimSpace(s.getenv(e.RootEnv));if root!=""{if !filepath.IsAbs(root){return "",fmt.Errorf("ADAPTER_INSTALL_FAILED: %s must be an absolute local path",e.RootEnv)};return root,nil}
	h,err:=s.home();if err!=nil{return "",err};if e.DefaultRootRelative==""||unsafeRel(e.DefaultRootRelative){return "",fmt.Errorf("ADAPTER_MANIFEST_INVALID: adapter install root is not configured")};return filepath.Join(h,filepath.FromSlash(e.DefaultRootRelative)),nil
}

func (s *AdapterService) Status(ctx context.Context,id string)(result.CommandResult,int){
	e,m,_,cleanup,err:=s.load(ctx,id);if cleanup!=nil{defer cleanup()};if err!=nil{return adapterFailure("adapter status",err)}
	root,err:=s.installRoot(e);if err!=nil{return adapterFailure("adapter status",err)}
	st:=AdapterStatus{Schema:AdapterStatusSchema,ID:id,Repository:e.Repository,RegistryStatus:e.Status,Compatible:adapterCompatible(m),Manifest:&m,Targets:[]string{}}
	for _,a:=range m.Assets{target,terr:=adapterTarget(root,e,a);if terr!=nil{return adapterFailure("adapter status",terr)};st.Targets=append(st.Targets,target);info,serr:=os.Stat(target);if serr==nil&&info.IsDir(){st.Installed=true;marker,ok:=readManagedMarker(target);st.Managed=st.Managed||ok&&marker.AdapterID==id&&marker.Repository==e.Repository}}
	r:=result.Success("adapter status",st);if !st.Compatible{r.Warnings=append(r.Warnings,"ADAPTER_INCOMPATIBLE");r.Constraints=append(r.Constraints,"adapter manifest does not declare the current iw v2 schemas")};return r,result.ExitOK
}

func (s *AdapterService) Doctor(ctx context.Context,id string)(result.CommandResult,int){r,code:=s.Status(ctx,id);if !r.OK{return r,code};st:=r.Data.(AdapterStatus);r.Command="adapter doctor";r.CanonicalCommand="adapter doctor";if !st.Compatible{r.OK=false;r.Error=&result.Error{Code:"ADAPTER_INCOMPATIBLE",Message:"adapter is not compatible with current iw protocol"};return r,result.ExitProtocol};return r,result.ExitOK}

func (s *AdapterService) Install(ctx context.Context,id string,dry bool)(result.CommandResult,int){return s.apply(ctx,"adapter install",id,dry,false)}
func (s *AdapterService) Update(ctx context.Context,id string,dry bool)(result.CommandResult,int){return s.apply(ctx,"adapter update",id,dry,true)}
func (s *AdapterService) apply(ctx context.Context,command,id string,dry,updating bool)(result.CommandResult,int){
	e,m,src,cleanup,err:=s.load(ctx,id);if cleanup!=nil{defer cleanup()};if err!=nil{return adapterFailure(command,err)};if !adapterCompatible(m){return adapterFailure(command,fmt.Errorf("ADAPTER_INCOMPATIBLE: adapter manifest does not require current iw v2 schemas"))}
	root,err:=s.installRoot(e);if err!=nil{return adapterFailure(command,err)};report:=AdapterActionReport{Status:AdapterStatus{Schema:AdapterStatusSchema,ID:id,Repository:e.Repository,RegistryStatus:e.Status,Compatible:true,Manifest:&m,Targets:[]string{}},DryRun:dry,Actions:[]string{}}
	for _,a:=range m.Assets{
		target,err:=adapterTarget(root,e,a);if err!=nil{return adapterFailure(command,err)};report.Status.Targets=append(report.Status.Targets,target)
		_,statErr:=os.Stat(target);exists:=statErr==nil
		if exists{marker,managed:=readManagedMarker(target);if !managed||marker.AdapterID!=id||marker.Repository!=e.Repository{return adapterFailure(command,fmt.Errorf("ADAPTER_UNMANAGED: refusing to overwrite existing unmanaged target %s",target))};report.Status.Installed=true;report.Status.Managed=true;if !updating{report.Actions=append(report.Actions,"already managed: "+target);continue}}
		report.Actions=append(report.Actions,map[bool]string{true:"update ",false:"install "}[updating]+target);if dry{continue}
		if err:=copyManagedAsset(filepath.Join(src,filepath.FromSlash(a.Source)),target,AdapterInstallMarker{Schema:AdapterInstallMarkerSchema,AdapterID:id,Repository:e.Repository,AssetKind:a.Kind,AssetSource:a.Source});err!=nil{return adapterFailure(command,fmt.Errorf("ADAPTER_INSTALL_FAILED: %w",err))};report.Status.Installed=true;report.Status.Managed=true
	}
	r:=result.Success(command,report);return r,result.ExitOK
}

func (s *AdapterService) Remove(ctx context.Context,id string,dry bool)(result.CommandResult,int){
	e,m,_,cleanup,err:=s.load(ctx,id);if cleanup!=nil{defer cleanup()};if err!=nil{return adapterFailure("adapter remove",err)};root,err:=s.installRoot(e);if err!=nil{return adapterFailure("adapter remove",err)}
	report:=AdapterActionReport{Status:AdapterStatus{Schema:AdapterStatusSchema,ID:id,Repository:e.Repository,RegistryStatus:e.Status,Compatible:adapterCompatible(m),Manifest:&m,Targets:[]string{}},DryRun:dry,Actions:[]string{}}
	for _,a:=range m.Assets{target,err:=adapterTarget(root,e,a);if err!=nil{return adapterFailure("adapter remove",err)};report.Status.Targets=append(report.Status.Targets,target);if _,err:=os.Stat(target);os.IsNotExist(err){continue};marker,managed:=readManagedMarker(target);if !managed||marker.AdapterID!=id||marker.Repository!=e.Repository{return adapterFailure("adapter remove",fmt.Errorf("ADAPTER_UNMANAGED: refusing to remove unmanaged target %s",target))};report.Actions=append(report.Actions,"remove "+target);if !dry{if err:=os.RemoveAll(target);err!=nil{return adapterFailure("adapter remove",fmt.Errorf("ADAPTER_INSTALL_FAILED: %w",err))}}}
	r:=result.Success("adapter remove",report);return r,result.ExitOK
}

func adapterTarget(root string,e AdapterRegistryEntry,a AdapterAsset)(string,error){tpl:=e.AssetTargets[a.Kind];rel:=strings.ReplaceAll(tpl,"{name}",a.Name);if unsafeRel(rel){return "",fmt.Errorf("ADAPTER_MANIFEST_INVALID: unsafe target %q",rel)};return filepath.Join(root,filepath.FromSlash(rel)),nil}
func unsafeRel(p string)bool{p=strings.TrimSpace(p);if p==""||filepath.IsAbs(p){return true};clean:=filepath.Clean(filepath.FromSlash(p));return clean==".."||strings.HasPrefix(clean,".."+string(filepath.Separator))}
func containsString(xs []string,w string)bool{for _,x:=range xs{if x==w{return true}};return false}
func sortStrings(xs []string){for i:=0;i<len(xs);i++{for j:=i+1;j<len(xs);j++{if xs[j]<xs[i]{xs[i],xs[j]=xs[j],xs[i]}}}}

func readManagedMarker(target string)(AdapterInstallMarker,bool){raw,err:=os.ReadFile(filepath.Join(target,".iw-adapter-install.json"));if err!=nil{return AdapterInstallMarker{},false};var m AdapterInstallMarker;if json.Unmarshal(raw,&m)!=nil||m.Schema!=AdapterInstallMarkerSchema{return AdapterInstallMarker{},false};return m,true}
func copyManagedAsset(source,target string,marker AdapterInstallMarker)error{
	info,err:=os.Lstat(source);if err!=nil{return err};if info.Mode()&os.ModeSymlink!=0{return fmt.Errorf("symlink source rejected")};if !info.IsDir(){return fmt.Errorf("adapter asset source must be a directory")}
	parent:=filepath.Dir(target);if err:=os.MkdirAll(parent,0755);err!=nil{return err};stage,err:=os.MkdirTemp(parent,".iw-stage-*");if err!=nil{return err};defer os.RemoveAll(stage)
	stageTarget:=filepath.Join(stage,"asset");if err:=copyDirNoSymlinks(source,stageTarget);err!=nil{return err};raw,_:=json.MarshalIndent(marker,"","  ");if err:=os.WriteFile(filepath.Join(stageTarget,".iw-adapter-install.json"),append(raw,'\n'),0644);err!=nil{return err}
	backup:="";if _,err:=os.Stat(target);err==nil{backup=target+".iw-backup";_ = os.RemoveAll(backup);if err:=os.Rename(target,backup);err!=nil{return err}}
	if err:=os.Rename(stageTarget,target);err!=nil{if backup!=""{_ = os.Rename(backup,target)};return err};if backup!=""{_ = os.RemoveAll(backup)};return nil
}
func copyDirNoSymlinks(src,dst string)error{return filepath.Walk(src,func(path string,info os.FileInfo,err error)error{if err!=nil{return err};if info.Mode()&os.ModeSymlink!=0{return fmt.Errorf("symlink rejected: %s",path)};rel,err:=filepath.Rel(src,path);if err!=nil{return err};target:=filepath.Join(dst,rel);if info.IsDir(){return os.MkdirAll(target,info.Mode().Perm())};in,err:=os.Open(path);if err!=nil{return err};defer in.Close();out,err:=os.OpenFile(target,os.O_CREATE|os.O_WRONLY|os.O_TRUNC,info.Mode().Perm());if err!=nil{return err};_,cpErr:=io.Copy(out,in);closeErr:=out.Close();if cpErr!=nil{return cpErr};return closeErr})}

func adapterFailure(command string,err error)(result.CommandResult,int){msg:=err.Error();code:="ADAPTER_INSTALL_FAILED";for _,candidate:=range []string{"ADAPTER_NOT_FOUND","ADAPTER_NOT_READY","ADAPTER_MANIFEST_INVALID","ADAPTER_INCOMPATIBLE","ADAPTER_UNMANAGED","ADAPTER_INSTALL_FAILED"}{if strings.HasPrefix(msg,candidate+":"){code=candidate;break}};return result.Failure(command,code,msg,nil,nil),result.ExitCodeFor(code)}
