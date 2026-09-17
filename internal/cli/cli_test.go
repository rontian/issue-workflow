package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rontian/issue-workflow/internal/result"
)

func TestVersionJSONIsSingleEnvelope(t *testing.T) {
	var out, errOut bytes.Buffer
	c := &CLI{Stdout: &out, Stderr: &errOut, Getwd: func() (string, error) { return "/tmp", nil }}
	code := c.Run(context.Background(), []string{"version", "--json"})
	if code != 0 { t.Fatalf("code=%d stderr=%s", code, errOut.String()) }
	var got result.CommandResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil { t.Fatal(err) }
	if got.Schema != result.Schema || !got.OK || got.Command != "version" { t.Fatalf("got=%+v", got) }
	if errOut.Len() != 0 { t.Fatalf("stderr=%q", errOut.String()) }
}
func TestCommonFlagBeforeCommand(t *testing.T) {
	var out bytes.Buffer
	c := &CLI{Stdout: &out, Stderr: &bytes.Buffer{}, Getwd: func() (string, error) { return "/tmp", nil }}
	code := c.Run(context.Background(), []string{"--json", "version"})
	if code != 0 { t.Fatalf("code=%d", code) }
	var got result.CommandResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil { t.Fatal(err) }
}
