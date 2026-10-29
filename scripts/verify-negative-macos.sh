#!/usr/bin/env bash
# Bash 3.2, isolated Core negative acceptance. No GitHub writes by default.
set -uo pipefail
export LC_ALL=C
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO=rontian/pi-test
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$HOME/Library/Logs/iw-verification/negative-$STAMP-$$"
if [ -n "${IW_VERIFY_REPORT_DIR:-}" ]; then OUT="$IW_VERIFY_REPORT_DIR/negative-$STAMP-$$"; fi
mkdir -p "$OUT/bin" || exit 1
printf 'status,check,exit_code,expected_error,actual_error,event_before,event_after\n' > "$OUT/results.csv"
LIVE=0; CLEANUP=0; FAILS=0; ISSUE=""; POISON=""; BRANCH=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) LIVE=1 ;;
    --cleanup) CLEANUP=1 ;;
    --help|-h) echo 'Usage: bash scripts/verify-negative-macos.sh [--live] [--cleanup]'; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done
if [ "$CLEANUP" -eq 1 ] && [ "$LIVE" -ne 1 ]; then echo '--cleanup requires --live' >&2; exit 2; fi
if [ "$LIVE" -eq 1 ] && [ "${IW_TEST_CONFIRM:-}" != YES ]; then echo 'Live requires IW_TEST_CONFIRM=YES' >&2; exit 2; fi
if [ "${IW_TEST_REPO:-rontian/pi-test}" != "$REPO" ]; then echo "Refusing non-test repository" >&2; exit 2; fi
record() {
  printf '%s,%s,%s,%s,%s,%s,%s\n' "$1" "$2" "$3" "$4" "$5" "$6" "$7" >> "$OUT/results.csv"
  echo "[$1] $2 (exit=$3, expected=$4, actual=$5, events=$6->$7)"
  if [ "$1" != PASS ]; then FAILS=$((FAILS+1)); fi
}
finish() {
  echo "Results: $OUT/results.csv"
  echo "Test Issue: $ISSUE; poison Issue: $POISON; branch: $BRANCH"
  echo "Fail count: $FAILS"
  echo 'Do not share unredacted logs or local absolute paths.'
  if [ "$FAILS" -ne 0 ]; then exit 1; fi
}
trap finish EXIT
local_check() {
  name="$1"; shift
  if "$@" > "$OUT/local-$name.log" 2>&1; then record PASS "$name" 0 '' '' '' ''
  else rc=$?; record FAIL "$name" "$rc" '' '' '' ''; tail -n 20 "$OUT/local-$name.log" >&2; exit 1; fi
}
for dependency in bash git go; do
  if ! command -v "$dependency" >/dev/null 2>&1; then record FAIL "missing-$dependency" 127 '' '' '' ''; exit 1; fi
done
local_check shell-syntax bash -n "$ROOT/scripts/verify-negative-macos.sh"
local_check unit bash -c 'cd "$1" && go test ./internal/domain ./internal/app ./internal/adapters/gitcli' _ "$ROOT"
local_check build bash -c 'cd "$1" && go build -o "$2" ./cmd/iw' _ "$ROOT" "$OUT/bin/iw"
if [ "$LIVE" -eq 0 ]; then echo 'Local PASS; no live GitHub access or writes.'; exit 0; fi
for dependency in gh jq; do
  if ! command -v "$dependency" >/dev/null 2>&1; then record FAIL "missing-$dependency" 127 '' '' '' ''; exit 1; fi
done
local_check auth gh auth status
local_check repo-identity bash -c 'gh repo view "$1" --json nameWithOwner --jq .nameWithOwner | grep -Fx "$1"' _ "$REPO"
BRANCH="iw-negative-$STAMP-$$"
WORK="$OUT/checkout"
# Network transport can fail before any writes. Retry only the sandbox clone,
# and remove only this invocation's partially created checkout.
CLONED=0
for attempt in 1 2 3; do
  if gh repo clone "$REPO" "$WORK" -- --depth 1 --config http.version=HTTP/1.1 > "$OUT/clone-$attempt.log" 2>&1; then
    CLONED=1
    break
  fi
  if [ -e "$WORK" ] || [ -L "$WORK" ]; then rm -rf "$WORK" || exit 1; fi
  if [ "$attempt" -lt 3 ]; then sleep "$attempt"; fi
done
if [ "$CLONED" -ne 1 ]; then record FAIL sandbox-clone 1 '' '' '' ''; exit 1; fi
if ! git -C "$WORK" switch -c "$BRANCH" > "$OUT/branch.log" 2>&1 ||
   ! git -C "$WORK" -c user.name=iw-negative -c user.email=iw-negative@users.noreply.github.com -c commit.gpgsign=false commit --allow-empty -m "[iw-negative] $BRANCH" > "$OUT/commit.log" 2>&1 ||
   ! git -C "$WORK" push -u origin "$BRANCH" > "$OUT/push.log" 2>&1; then
  record FAIL sandbox-setup 1 '' '' '' ''; exit 1
fi
record PASS sandbox-setup 0 '' '' '' ''
cd "$WORK" || exit 1
iw() { "$OUT/bin/iw" "$@" --json; }
events() {
  gh api --paginate --slurp "repos/$REPO/issues/$1/comments?per_page=100" |
    jq -er '[.[][] | select(.body | contains("<!-- iw:workflow-event:"))] | length'
}
positive() {
  name="$1"; shift
  if iw "$@" > "$OUT/$name.json" 2> "$OUT/$name.stderr" &&
     jq -e '.ok == true and .schema == "iw.command-result/v2"' "$OUT/$name.json" >/dev/null; then
    record PASS "$name" 0 '' '' '' ''
  else
    rc=$?; record FAIL "$name" "$rc" '' '' '' ''
    jq -c '{error,command}' "$OUT/$name.json" >&2 2>/dev/null || :
    exit 1
  fi
}
negative() {
  name="$1"; target="$2"; expected="$3"; shift 3
  before="$(events "$target")" || { record FAIL "$name-count-before" 1 "$expected" '' '' ''; exit 1; }
  iw "$@" > "$OUT/$name.json" 2> "$OUT/$name.stderr"
  rc=$?
  after="$(events "$target")" || { record FAIL "$name-count-after" 1 "$expected" '' "$before" ''; exit 1; }
  actual="$(jq -r '.error.code // empty' "$OUT/$name.json" 2>/dev/null)"
  if [ "$rc" -ne 0 ] && [ "$actual" = "$expected" ] && [ "$before" = "$after" ] &&
     jq -e '.ok == false and .schema == "iw.command-result/v2" and .machine.schema == "iw.machine-semantics/v1"' "$OUT/$name.json" >/dev/null; then
    record PASS "$name" "$rc" "$expected" "$actual" "$before" "$after"
  else record FAIL "$name" "$rc" "$expected" "$actual" "$before" "$after"; exit 1; fi
}
# Never retry an ambiguous create: otherwise it could duplicate test Issues.
positive create new --title "[iw-negative] $BRANCH" --goal 'Synthetic fail-closed acceptance' \
  --scope 'Only this isolated Issue and branch' --out-of-scope 'All existing repository data' \
  --acceptance 'Rejected operations append no event' --validation 'true' \
  --operation-id "$BRANCH-create"
ISSUE="$(jq -r '.issue // empty' "$OUT/create.json")"
case "$ISSUE" in ''|*[!0-9]*) record FAIL issue-number 1 '' '' '' ''; exit 1 ;; esac
positive start start "$ISSUE" --operation-id "$BRANCH-start"
positive checkpoint checkpoint "$ISSUE" --summary 'Initial synthetic validation' --next 'negative gates' \
  --validation 'v1=pass:synthetic check' --operation-id "$BRANCH-cp"
positive review review "$ISSUE" --summary 'Initial synthetic review passed' --operation-id "$BRANCH-review"
negative operation-conflict "$ISSUE" CONFLICT start "$ISSUE" \
  --summary 'different payload' --operation-id "$BRANCH-start"
before="$(events "$ISSUE")" || exit 1
positive dry-run pause "$ISSUE" --reason 'preview only' --dry-run --operation-id "$BRANCH-preview"
after="$(events "$ISSUE")" || exit 1
if [ "$before" = "$after" ]; then record PASS dry-run-no-event 0 '' '' "$before" "$after"
else record FAIL dry-run-no-event 1 '' '' "$before" "$after"; exit 1; fi

gh issue view "$ISSUE" --repo "$REPO" --json body --jq .body > "$OUT/original-body.md" || exit 1
cp "$OUT/original-body.md" "$OUT/changed-body.md" || exit 1
printf '\n<!-- iw-negative contract drift %s -->\n' "$BRANCH" >> "$OUT/changed-body.md"
if ! gh issue edit "$ISSUE" --repo "$REPO" --body-file "$OUT/changed-body.md" > "$OUT/edit.log" 2>&1; then
  record FAIL contract-edit 1 '' '' '' ''; exit 1
fi
negative contract-drift "$ISSUE" CONTRACT_CHANGED checkpoint "$ISSUE" \
  --summary 'reject changed contract' --next 'no event' --operation-id "$BRANCH-drift"
positive scope-accept scope accept "$ISSUE" --reason 'Approved isolated synthetic contract drift' \
  --operation-id "$BRANCH-scope"
positive scope-status status "$ISSUE"
if jq -e '.data.contract_changed == false' "$OUT/scope-status.json" >/dev/null; then
  record PASS scope-digest-reconciled 0 '' '' '' ''
else record FAIL scope-digest-reconciled 1 '' '' '' ''; exit 1; fi
# Scope acceptance invalidates prior review. Establish a fresh passing review
# at the original HEAD so the next HEAD change exercises stale-review gating.
positive review-after-scope review "$ISSUE" --summary 'Synthetic review after contract reconciliation' \
  --operation-id "$BRANCH-review-after-scope"

printf 'untracked synthetic content\n' > "$WORK/iw-negative-dirty.txt"
negative dirty-worktree "$ISSUE" STALE_GIT_STATE handoff "$ISSUE" \
  --summary 'refuse dirty' --next 'no event' --portable --operation-id "$BRANCH-dirty"
rm "$WORK/iw-negative-dirty.txt" || exit 1
if ! git -C "$WORK" -c user.name=iw-negative -c user.email=iw-negative@users.noreply.github.com -c commit.gpgsign=false commit --allow-empty -m "[iw-negative] HEAD drift" > "$OUT/head-drift.log" 2>&1; then
  record FAIL commit-head-drift 1 '' '' '' ''; exit 1
fi
negative git-drift "$ISSUE" STALE_GIT_STATE complete "$ISSUE" \
  --summary 'refuse stale Git HEAD' --operation-id "$BRANCH-git"
negative unpublished-handoff "$ISSUE" STALE_GIT_STATE handoff "$ISSUE" \
  --summary 'refuse unpublished HEAD' --next 'no event' --portable --operation-id "$BRANCH-unpublished"
if ! git -C "$WORK" push origin "$BRANCH" > "$OUT/head-push.log" 2>&1; then
  record FAIL push-head 1 '' '' '' ''; exit 1
fi
positive checkpoint-head checkpoint "$ISSUE" --summary 'New HEAD no validation' --next 'review' \
  --operation-id "$BRANCH-head-cp"
negative stale-review "$ISSUE" REVIEW_REQUIRED complete "$ISSUE" \
  --summary 'reject stale review' --operation-id "$BRANCH-stale-review"
positive review-head review "$ISSUE" --summary 'Reviewed new HEAD' --operation-id "$BRANCH-review-head"
negative stale-validation "$ISSUE" VALIDATION_FAILED complete "$ISSUE" \
  --summary 'reject stale validation' --operation-id "$BRANCH-stale-val"
positive checkpoint-validation checkpoint "$ISSUE" --summary 'Synthetic current validation' \
  --next 'handoff' --validation 'v1=pass:synthetic current HEAD' --operation-id "$BRANCH-val-head"
positive handoff handoff "$ISSUE" --summary 'Exact published HEAD' --next 'restore on other host' \
  --portable --operation-id "$BRANCH-handoff"
if ! git -C "$WORK" -c user.name=iw-negative -c user.email=iw-negative@users.noreply.github.com -c commit.gpgsign=false commit --allow-empty -m "[iw-negative] remote moved" > "$OUT/remote-change.log" 2>&1 ||
   ! git -C "$WORK" push origin "$BRANCH" > "$OUT/remote-push.log" 2>&1; then
  record FAIL remote-change 1 '' '' '' ''; exit 1
fi
HEAD_BEFORE="$(git -C "$WORK" rev-parse HEAD)" || exit 1
negative restore-mismatch "$ISSUE" RESTORE_CONFLICT restore "$ISSUE" --apply
HEAD_AFTER="$(git -C "$WORK" rev-parse HEAD)" || exit 1
if [ "$HEAD_BEFORE" = "$HEAD_AFTER" ]; then record PASS restore-no-reset 0 '' '' '' ''
else record FAIL restore-no-reset 1 '' '' '' ''; exit 1; fi

# Separate, freshly created sacrificial Issue for malformed workflow comments.
positive poison-create new --title "[iw-negative] poison-$BRANCH" \
  --goal 'Malformed schema rejection' --scope 'This isolated Issue only' \
  --out-of-scope 'Any other Issue' --acceptance 'Reject unknown events' \
  --operation-id "$BRANCH-poison-create"
POISON="$(jq -r '.issue // empty' "$OUT/poison-create.json")"
case "$POISON" in ''|*[!0-9]*) record FAIL poison-number 1 '' '' '' ''; exit 1 ;; esac
printf '<!-- iw:workflow-event:v99\n{}\n-->\n' > "$OUT/poison.md"
if ! gh issue comment "$POISON" --repo "$REPO" --body-file "$OUT/poison.md" > "$OUT/poison-comment.log" 2>&1; then
  record FAIL poison-comment 1 '' '' '' ''; exit 1
fi
negative unknown-schema "$POISON" UNSUPPORTED_SCHEMA start "$POISON" \
  --operation-id "$BRANCH-poison-start"
negative unknown-schema-read "$POISON" UNSUPPORTED_SCHEMA status "$POISON"

positive cancel cancel "$ISSUE" --reason 'Synthetic negative acceptance completed' \
  --operation-id "$BRANCH-cancel"
positive close issue close "$ISSUE"
if ! gh issue close "$POISON" --repo "$REPO" --reason 'not planned' > "$OUT/poison-close.log" 2>&1; then
  record FAIL poison-close 1 '' '' '' ''; exit 1
fi
record PASS poison-close 0 '' '' '' ''
if [ "$CLEANUP" -eq 1 ]; then
  if ! git -C "$WORK" push origin ":refs/heads/$BRANCH" > "$OUT/cleanup.log" 2>&1; then
    record FAIL branch-cleanup 1 '' '' '' ''; exit 1
  fi
  record PASS branch-cleanup 0 '' '' '' ''
fi
