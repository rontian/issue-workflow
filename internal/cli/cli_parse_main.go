package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func parse(args []string) (string, options, bool, error) {
	var o options
	pre := flag.NewFlagSet("iw", flag.ContinueOnError)
	pre.SetOutput(io.Discard)
	registerFlags(pre, &o)
	if e := pre.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) { return "", o, true, nil }
		return "", o, false, e
	}
	rest := pre.Args()
	if len(rest) == 0 { return "", o, false, errors.New("缺少 command") }
	cmd := rest[0]
	if cmd == "help" {
		if len(rest) > 1 { return rest[1], o, true, nil }
		return "", o, true, nil
	}
	ordered, e := flagsBeforePositionals(rest[1:])
	if e != nil { return cmd, o, false, e }
	tail := flag.NewFlagSet(cmd, flag.ContinueOnError)
	tail.SetOutput(io.Discard)
	registerFlags(tail, &o)
	if e := tail.Parse(ordered); e != nil {
		if errors.Is(e, flag.ErrHelp) { return cmd, o, true, nil }
		return cmd, o, false, e
	}
	pos := tail.Args()
	switch cmd {
	case "version", "doctor", "init", "new":
		if len(pos) != 0 { return cmd, o, false, fmt.Errorf("unexpected positional arguments: %s", strings.Join(pos, " ")) }
	case "scope":
		if len(pos) != 2 { return cmd, o, false, errors.New("Usage: iw scope propose|accept|reject <issue>") }
		a := strings.ToLower(pos[0])
		if a != "propose" && a != "accept" && a != "reject" { return cmd, o, false, errors.New("scope action must be propose|accept|reject") }
		o.ScopeAction = a
		n, e := strconv.Atoi(pos[1]); if e != nil || n <= 0 { return cmd, o, false, errors.New("Issue number 必须是正整数") }
		if o.Issue > 0 && o.Issue != n { return cmd, o, false, errors.New("positional Issue 与 --issue 不一致") }
		o.Issue = n
	case "status", "resume", "start", "checkpoint", "handoff", "block", "review", "fix", "final":
		if len(pos) > 1 { return cmd, o, false, errors.New("只允许一个 Issue number") }
		if len(pos) == 1 {
			n, e := strconv.Atoi(pos[0]); if e != nil || n <= 0 { return cmd, o, false, errors.New("Issue number 必须是正整数") }
			if o.Issue > 0 && o.Issue != n { return cmd, o, false, errors.New("positional Issue 与 --issue 不一致") }
			o.Issue = n
		}
		if o.Issue <= 0 { return cmd, o, false, errors.New("缺少 Issue number") }
	default:
		return cmd, o, false, fmt.Errorf("unknown command %q", cmd)
	}
	return cmd, o, false, nil
}
