#!/usr/bin/env bash
# Portable macOS bash 3.2; no Homebrew GNU utilities required.
# Usage:
#   bash scripts/verify-macos.sh
#   IW_TEST_CONFIRM=YES bash scripts/verify-macos.sh --live
#   IW_TEST_CONFIRM=YES bash scripts/verify-macos.sh --live --cleanup
set -uo pipefail
export LC_ALL=C

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
REPORT_BASE="${IW_VERIFY_REPORT_DIR:-${HOME}/Library/Logs/iw-verification}"
REPORT_DIR="${REPORT_BASE%/}/${STAMP}-$$"
mkdir -p "$REPORT_DIR" || exit 1
BIN_DIR="$REPORT_DIR/bin"
mkdir -p "$BIN_DIR"
TARGET_REPO="rontian/pi-test"
LIVE=0
CLEANUP=0
FAILURES=0
ISSUE=""
TEST_BRANCH=""
CHECKOUT_A=""
CHECKOUT_B=""
STAGE="local"
printf '%s\n' "Verification output: $REPORT_DIR"
printf 'status,stage,check,exit_code\n' > "$REPORT_DIR/results.csv"
printf 'date_utc=%s\nroot=%s\n' "$STAMP" "$ROOT" > "$REPORT_DIR/meta.txt"
git -C "$ROOT" rev-parse HEAD >> "$REPORT_DIR/meta.txt" 2>/dev/null || :
git -C "$ROOT" branch --show-current >> "$REPORT_DIR/meta.txt" 2>/dev/null || :

usage() {
  echo "Usage: bash scripts/verify-macos.sh [--live] [--cleanup]"
  echo "Default: local Go tests, race, vet, build, install and CLI smoke (no GitHub writes)."
  echo "--live requires IW_TEST_CONFIRM=YES; writes only to test branch and fresh [iw-blackbox] Issue in rontian/pi-test."
  echo "--cleanup can be combined with --live, closes ONLY the newly created Issue and deletes ONLY the newly created test branch."
}

for arg in "$@"; do
  case "$arg" in
    --live) LIVE=1 ;;
    --cleanup) CLEANUP=1 ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done
if [ "$CLEANUP" -eq 1 ] && [ "$LIVE" -ne 1 ]; then
  echo "--cleanup requires --live" >&2; exit 2
fi
if [ "$LIVE" -eq 1 ] && [ "${IW_TEST_CONFIRM:-}" != YES ]; then
  echo "Live test is disabled. Set IW_TEST_CONFIRM=YES explicitly." >&2; exit 2
fi

record() {
  status="$1"; check="$2"; code="$3"
  printf '%s,%s,%s,%s\n' "$status" "$STAGE" "$check" "$code" >> "$REPORT_DIR/results.csv"
  echo "[$status] $STAGE/$check (exit $code)"
  if [ "$status" != PASS ]; then FAILURES=$((FAILURES+1)); fi
}
run() {
  check="$1"; shift
  log="$REPORT_DIR/${STAGE}-${check}.log"
  echo ">> $STAGE/$check"
  if "$@" > "$log" 2>&1; then
    record PASS "$check" 0
  else
    rc=$?
    record FAIL "$check" "$rc"
    tail -n 35 "$log" >&2
  fi
}
require_command() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; record FAIL "missing-$1" 127; return 1; }
}
assert_json() {
  echo "$1" | jq -e "$2" >/dev/null 2>&1
}
live_iw() {
  # Logs contain test Issue data. Do not forward unreviewed logs from private repositories.
  "$BIN_DIR/iw" "$@" --json
}
run_live_step() {
  name="$1"; shift
  log="$REPORT_DIR/live-$name.json"
  if "$@" > "$log" 2> "$REPORT_DIR/live-$name.stderr"; then
    if jq -e '.ok == true and .schema == "iw.command-result/v2"' "$log" >/dev/null; then
      record PASS "$name" 0; return 0
    fi
    record FAIL "$name" 1
  else
    rc=$?
    record FAIL "$name" "$rc"
  fi
  tail -n 20 "$log" 2>/dev/null >&2 || :
  return 1
}
finish() {
  STAGE=local
  echo "----------------------------------------"
  echo "Results: $REPORT_DIR/results.csv"
  echo "Repo state: $REPORT_DIR/meta.txt"
  echo "Created live Issue: ${ISSUE:-none}"
  echo "Created test branch: ${TEST_BRANCH:-none}"
  echo "Fail count: $FAILURES"
  echo "Send results.csv + failing step logs after removing any private information."
  if [ "$FAILURES" -ne 0 ]; then exit 1; fi
}
trap finish EXIT

STAGE=local
require_command go || exit 1
require_command git || exit 1
echo "Go: $(go version)"
run shell-syntax bash -n "$ROOT/scripts/verify-macos.sh"
# Compile all Go packages without running tests; unlike gofmt, this verifies only compilable source.
run go-compile bash -c 'cd "$1" && go test -run "^$" ./...' _ "$ROOT"
run unit bash -c 'cd "$1" && go test ./...' _ "$ROOT"
run race bash -c 'cd "$1" && go test -race ./...' _ "$ROOT"
run vet bash -c 'cd "$1" && go vet ./...' _ "$ROOT"
run build bash -c 'cd "$1" && go build -o "$2" ./cmd/iw' _ "$ROOT" "$BIN_DIR/iw"
run install bash -c 'cd "$1" && GOBIN="$2" go install ./cmd/iw' _ "$ROOT" "$BIN_DIR"

if [ -x "$BIN_DIR/iw" ]; then
  run version-schema bash -c '"$1" version --json | grep -q '"'"'"schema":"iw.command-result/v2"'"'"'' _ "$BIN_DIR/iw"
  run capability-schema bash -c '"$1" capabilities --json | grep -q '"'"'"schema":"iw.capabilities/v1"'"'"'' _ "$BIN_DIR/iw"
  run help bash -c '"$1" --help | grep -q "iw"' _ "$BIN_DIR/iw"
fi
if [ "$FAILURES" -ne 0 ]; then
  echo "Local validation failed; skipping any live GitHub changes." >&2
  exit 1
fi
if [ "$LIVE" -eq 0 ]; then
  echo "Local-only validation completed. Run with --live after reviewing the test isolation policy."
  exit 0
fi

STAGE=live
require_command gh || exit 1
require_command jq || exit 1
# Never run black-box tests in Core or .pi; use only the dedicated sandbox.
if [ "${IW_TEST_REPO:-rontian/pi-test}" != "$TARGET_REPO" ]; then
  echo "Safety guard: live repository must be exactly $TARGET_REPO" >&2
  record FAIL wrong-repository 2; exit 1
fi
run auth bash -c 'gh auth status >/dev/null 2>&1'
if [ "$FAILURES" -ne 0 ]; then exit 1; fi

TEST_BRANCH="iw-blackbox-${STAMP}-$$"
CHECKOUT_A="$REPORT_DIR/checkout-a"
CHECKOUT_B="$REPORT_DIR/checkout-b"
run clone-a gh repo clone "$TARGET_REPO" "$CHECKOUT_A" -- --depth 1
if [ ! -d "$CHECKOUT_A/.git" ]; then exit 1; fi
run branch-a git -C "$CHECKOUT_A" switch -c "$TEST_BRANCH"
run commit-a git -C "$CHECKOUT_A" -c user.name="iw-blackbox" -c user.email="iw-blackbox@users.noreply.github.com" -c commit.gpgsign=false commit --allow-empty -m "[iw-blackbox] isolated workflow HEAD $STAMP"
run publish-a git -C "$CHECKOUT_A" push -u origin "$TEST_BRANCH"
if [ "$FAILURES" -ne 0 ]; then exit 1; fi

cd "$CHECKOUT_A" || exit 1
OP="iw-blackbox-${STAMP}-$$"
if ! run_live_step create live_iw new --title "[iw-blackbox] $OP" --goal "Verify iw with a dedicated test-only issue" --scope "Exercise the Core lifecycle" --out-of-scope "Modify unrelated repository data" --acceptance "The workflow is verified" --task "T01: Synthetic first milestone" --task "T02: Synthetic later milestone" --validation "true" --operation-id "$OP"; then exit 1; fi
ISSUE="$(jq -r '.issue // empty' "$REPORT_DIR/live-create.json")"
case "$ISSUE" in ''|*[!0-9]*) echo "Could not identify created Issue" >&2;record FAIL issue-number 1;exit 1;; esac
echo "Test Issue: https://github.com/$TARGET_REPO/issues/$ISSUE"
run_live_step doctor live_iw doctor --issue "$ISSUE" || exit 1
run_live_step start live_iw start "$ISSUE" --operation-id "$OP-start" || exit 1
# Retry of the exact same logical operation must not create a second event.
run_live_step start-idempotent live_iw start "$ISSUE" --operation-id "$OP-start" || exit 1
if ! jq -e '(.warnings | index("IDEMPOTENT_REPLAY")) != null' "$REPORT_DIR/live-start-idempotent.json" >/dev/null; then record FAIL start-idempotency 1;exit 1;fi
record PASS start-idempotency 0
run_live_step checkpoint live_iw checkpoint "$ISSUE" --summary "Safe synthetic milestone" --next "test external wait" --validation "v1=pass:synthetic true check" --operation-id "$OP-checkpoint" || exit 1
run_live_step task-status-a live_iw task status "$ISSUE" || exit 1
# Completing while T01/T02 are incomplete must be denied without a remote event.
if live_iw complete "$ISSUE" --summary "must fail before implementation" > "$REPORT_DIR/live-negative-early-complete.json" 2> "$REPORT_DIR/live-negative-early-complete.stderr"; then
  record FAIL early-complete-rejected 1; exit 1
elif jq -e '.ok == false and .error.code == "TRANSITION_BLOCKED"' "$REPORT_DIR/live-negative-early-complete.json" >/dev/null; then
  record PASS early-complete-rejected 0
else
  record FAIL early-complete-rejected 1; exit 1
fi
run_live_step task-one live_iw task complete "$ISSUE" --task-id T01 --summary "Synthetic implementation T01 completed" --operation-id "$OP-t1" || exit 1

run_live_step wait live_iw wait "$ISSUE" --reason "CI and manual condition" --next "recheck both" --condition ci=waiting:synthetic-pending --condition owner=waiting:synthetic-pending --operation-id "$OP-wait" || exit 1
# Blind aggregate resolution cannot bypass known individual conditions.
if live_iw recheck "$ISSUE" --resolved > "$REPORT_DIR/live-negative-blind-recheck.json" 2> "$REPORT_DIR/live-negative-blind-recheck.stderr"; then
  record FAIL blind-recheck-rejected 1; exit 1
elif jq -e '.ok == false and .error.code == "INVALID_INVOCATION"' "$REPORT_DIR/live-negative-blind-recheck.json" >/dev/null; then
  record PASS blind-recheck-rejected 0
else
  record FAIL blind-recheck-rejected 1; exit 1
fi
run_live_step recheck-partial live_iw recheck "$ISSUE" --condition ci=resolved:synthetic-test-succeeded --condition owner=waiting:synthetic-pending --operation-id "$OP-partial" || exit 1
if ! jq -e '.lifecycle == "WAITING"' "$REPORT_DIR/live-recheck-partial.json" >/dev/null; then record FAIL partial-stays-waiting 1;exit 1;fi
record PASS partial-stays-waiting 0
run_live_step recheck-all live_iw recheck "$ISSUE" --condition ci=resolved:synthetic-test-succeeded --condition owner=resolved:synthetic-owner-approved --operation-id "$OP-recheck" || exit 1
if ! jq -e '.lifecycle == "IN_PROGRESS"' "$REPORT_DIR/live-recheck-all.json" >/dev/null; then record FAIL all-exits-waiting 1;exit 1;fi
record PASS all-exits-waiting 0

run_live_step settle-later-task live_iw settle "$ISSUE" --item-id task:T02 --kind implementation --fact pending --disposition deferred --target defer --reason "Move synthetic T02 into the next explicit milestone" --operation-id "$OP-settle-t2" || exit 1
run_live_step defer live_iw defer "$ISSUE" --reason "later milestone" --next "resume test" --operation-id "$OP-defer" || exit 1
if ! jq -e '.lifecycle == "DEFERRED"' "$REPORT_DIR/live-defer.json" >/dev/null; then record FAIL deferred-lifecycle 1;exit 1;fi
record PASS deferred-lifecycle 0
run_live_step resume live_iw resume "$ISSUE" --summary "resumed in test" --operation-id "$OP-resume" || exit 1
run_live_step task-two live_iw task complete "$ISSUE" --task-id T02 --summary "Synthetic implementation T02 completed after resume" --operation-id "$OP-t2" || exit 1
run_live_step handoff live_iw handoff "$ISSUE" --summary "published exact HEAD" --next "restore in checkout-b" --portable --operation-id "$OP-handoff" || exit 1

cd "$REPORT_DIR" || exit 1
run clone-b gh repo clone "$TARGET_REPO" "$CHECKOUT_B" -- --depth 1
if [ ! -d "$CHECKOUT_B/.git" ]; then exit 1; fi
cd "$CHECKOUT_B" || exit 1
run_live_step restore-preview live_iw restore "$ISSUE" || exit 1
run_live_step restore-apply live_iw restore "$ISSUE" --apply || exit 1
run_live_step context-b live_iw context "$ISSUE" || exit 1
if ! jq -e '.data.task_progress.all_completed == true and (.data.settlement.items | length) == 0' "$REPORT_DIR/live-context-b.json" >/dev/null; then record FAIL segment-facts-restored 1;exit 1;fi
record PASS segment-facts-restored 0
if ! jq -e '.data.schema == "iw.context/v1"' "$REPORT_DIR/live-context-b.json" >/dev/null; then record FAIL portable-context-schema 1;exit 1;fi
record PASS portable-context-schema 0

run_live_step review live_iw review "$ISSUE" --summary "Synthetic black-box review passed" --operation-id "$OP-review" || exit 1
run_live_step complete live_iw complete "$ISSUE" --summary "Synthetic end-to-end workflow passed" --operation-id "$OP-complete" || exit 1
if ! jq -e '.lifecycle == "COMPLETED"' "$REPORT_DIR/live-complete.json" >/dev/null; then record FAIL completed-lifecycle 1;exit 1;fi
record PASS completed-lifecycle 0

# Native GitHub Issue close is explicit and independently confirmed by iw.
run_live_step native-close-completed live_iw issue close "$ISSUE" || exit 1
if ! jq -e '.data.desired_state == "CLOSED" and .data.applied == true' "$REPORT_DIR/live-native-close-completed.json" >/dev/null; then record FAIL github-close-confirmation 1;exit 1;fi
record PASS github-close-confirmation 0

# Cancel/reopen are distinct from completed and must preserve native state truth.
run_live_step workflow-reopen live_iw reopen "$ISSUE" --reason "Synthetic black-box second segment" --operation-id "$OP-reopen" || exit 1
run_live_step native-reopen live_iw issue reopen "$ISSUE" || exit 1
run_live_step cancel live_iw cancel "$ISSUE" --reason "Black-box second segment intentionally cancelled" --operation-id "$OP-cancel" || exit 1
run_live_step native-close-cancelled live_iw issue close "$ISSUE" || exit 1

if [ "$CLEANUP" -eq 1 ]; then
  # Only clean artifacts this invocation created; do not delete pre-existing data.
  echo "Test Issue has already been closed by iw; GitHub Issue history is intentionally retained for audit."
  run cleanup-branch git -C "$CHECKOUT_A" push origin ":refs/heads/$TEST_BRANCH"
else
  echo "Cleanup not requested. Only this run's test Issue and branch remain:"
  echo "  https://github.com/$TARGET_REPO/issues/$ISSUE"
  echo "  $TEST_BRANCH"
fi
