package gitcli

import (
	"reflect"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	for _, tc := range []struct{ in, host, full string }{{"git@github.com:rontian/issue-workflow.git", "github.com", "rontian/issue-workflow"}, {"https://github.com/rontian/issue-workflow.git", "github.com", "rontian/issue-workflow"}} {
		g, e := ParseRemoteURL(tc.in)
		if e != nil || g.Host != tc.host || g.FullName() != tc.full { t.Fatalf("%q => %+v %v", tc.in, g, e) }
	}
}
func TestPorcelain(t *testing.T) {
	got, e := ParsePorcelainV2Z([]byte("? z.txt\x001 .M N... 100644 100644 100644 abc abc a.txt\x00"))
	if e != nil { t.Fatal(e) }
	if !reflect.DeepEqual(got, []string{"a.txt", "z.txt"}) { t.Fatalf("%v", got) }
}
