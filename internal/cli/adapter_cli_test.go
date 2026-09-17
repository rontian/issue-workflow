package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/result"
)

func TestParseAdapterLifecycle(t *testing.T) {
	cmd, o, _, err := parse([]string{"adapter", "install", "codex", "--dry-run", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "adapter install" || o.AdapterID != "codex" || !o.DryRun || !o.JSON {
		t.Fatalf("%s %#v", cmd, o)
	}

	cmd, o, _, err = parse([]string{"adapter", "list", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "adapter list" || o.AdapterID != "" || !o.JSON {
		t.Fatalf("%s %#v", cmd, o)
	}
}

func TestParseDoctorAndInitAdapterFlag(t *testing.T) {
	cmd, o, _, err := parse([]string{"doctor", "--adapter", "codex", "--json"})
	if err != nil || cmd != "doctor" || o.AdapterID != "codex" || !o.JSON {
		t.Fatalf("cmd=%s options=%#v err=%v", cmd, o, err)
	}
	cmd, o, _, err = parse([]string{"init", "--adapter", "codex", "--dry-run"})
	if err != nil || cmd != "init" || o.AdapterID != "codex" || !o.DryRun {
		t.Fatalf("cmd=%s options=%#v err=%v", cmd, o, err)
	}
}

func TestAdapterListJSONSmoke(t *testing.T) {
	var out, errOut bytes.Buffer
	c := &CLI{
		Adapter: &app.AdapterService{},
		Stdout:  &out,
		Stderr:  &errOut,
		Getwd:   func() (string, error) { return "/tmp", nil },
	}
	code := c.Run(context.Background(), []string{"adapter", "list", "--json"})
	if code != result.ExitOK {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	var got result.CommandResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Command != "adapter list" || got.Schema != result.Schema {
		t.Fatalf("result=%#v", got)
	}
}

func TestPlannedAdapterFailsBeforeInstallPathGuessing(t *testing.T) {
	var out bytes.Buffer
	c := &CLI{
		Adapter: &app.AdapterService{},
		Stdout:  &out,
		Stderr:  &bytes.Buffer{},
		Getwd:   func() (string, error) { return "/tmp", nil },
	}
	code := c.Run(context.Background(), []string{"adapter", "install", "pi", "--dry-run", "--json"})
	if code != result.ExitInvalid {
		t.Fatalf("code=%d output=%s", code, out.String())
	}
	var got result.CommandResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OK || got.Error == nil || got.Error.Code != "ADAPTER_NOT_READY" {
		t.Fatalf("result=%#v", got)
	}
}

func TestUsageCoversV2CommandSurface(t *testing.T) {
	var out bytes.Buffer
	c := &CLI{}
	c.printUsage(&out)
	text := out.String()
	for _, want := range []string{
		"iw tasks",
		"iw next",
		"iw migrate <issue>",
		"iw pause <issue>",
		"iw wait <issue>",
		"iw recheck <issue>",
		"iw defer <issue>",
		"iw restore <issue> [--apply]",
		"iw complete <issue>",
		"iw cancel <issue>",
		"iw reopen <issue>",
		"iw adapter install <id> [--dry-run]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage missing %q\n%s", want, text)
		}
	}
}
