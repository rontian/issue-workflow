package cli

import "github.com/rontian/issue-workflow/internal/domain"

func parseValidationRecords(xs []string) ([]domain.ValidationRecord, error) {
	out := []domain.ValidationRecord{}
	for _, x := range xs {
		r, e := domain.ParseValidationRecord(x)
		if e != nil { return nil, e }
		out = append(out, r)
	}
	return out, nil
}

func commandOrUnknown(s string) string { if s == "" { return "unknown" }; return s }
func containsJSONFlag(args []string) bool { for _, a := range args { if a == "--json" { return true } }; return false }
