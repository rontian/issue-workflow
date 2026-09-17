package app

import (
	"sort"

	"github.com/rontian/issue-workflow/internal/domain"
)

type CheckStatus string
const(CheckPass CheckStatus="PASS";CheckWarn CheckStatus="WARN";CheckFail CheckStatus="FAIL";CheckSkip CheckStatus="SKIP")
type Check struct{ID string `json:"id"`;Category string `json:"category"`;Status CheckStatus `json:"status"`;Required bool `json:"required"`;Message string `json:"message"`;Next string `json:"next,omitempty"`}
type CheckSummary struct{Pass int `json:"pass"`;Warn int `json:"warn"`;Fail int `json:"fail"`;Skip int `json:"skip"`}
type DoctorReport struct{Repository *domain.RepositoryIdentity `json:"repository,omitempty"`;Checks []Check `json:"checks"`;Summary CheckSummary `json:"summary"`}
func summarize(cs []Check)CheckSummary{var s CheckSummary;for _,c:=range cs{switch c.Status{case CheckPass:s.Pass++;case CheckWarn:s.Warn++;case CheckFail:s.Fail++;case CheckSkip:s.Skip++}};return s}
func sortChecks(cs []Check){ord:=map[string]int{"dependency":0,"git":1,"github":2,"project":3,"workflow":4};sort.SliceStable(cs,func(i,j int)bool{oi,ok:=ord[cs[i].Category];if !ok{oi=99};oj,ok:=ord[cs[j].Category];if !ok{oj=99};if oi==oj{return cs[i].ID<cs[j].ID};return oi<oj})}
func firstRequiredFailure(cs []Check)*Check{for i:=range cs{if cs[i].Required&&cs[i].Status==CheckFail{return &cs[i]}};return nil}
