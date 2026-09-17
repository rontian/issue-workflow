package app

import "testing"

func TestSummarize(t *testing.T) {
	s := summarize([]Check{{Status: CheckPass}, {Status: CheckFail}, {Status: CheckWarn}, {Status: CheckSkip}})
	if s.Pass != 1 || s.Fail != 1 || s.Warn != 1 || s.Skip != 1 { t.Fatal(s) }
}
