#!/usr/bin/env bash
# Security sweeps: no-token -> 401, traversal -> 400, PIN gate.
# Usage: LANBOX_TOKEN=<token> LANBOX_PIN=<pin> ./test/security.sh [base]
# Exits non-zero on first failure.
set -u
BASE="${1:-http://localhost:8080}"
fail() { echo "FAIL: $1"; exit 1; }

echo "== no-token sweep (want 401) =="
for ep in "info" "files?path=/" "files/download?path=/x"; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/$ep") || fail "curl $ep"
  [ "$code" = "401" ] || fail "$ep -> $code (want 401)"
  echo "OK $ep -> 401"
done
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/files/upload?path=/") || fail "curl upload"
[ "$code" = "401" ] || fail "upload -> $code (want 401)"
echo "OK upload -> 401"

echo "== traversal sweep (want 400, needs LANBOX_TOKEN + LANBOX_PIN) =="
: "${LANBOX_TOKEN:?set LANBOX_TOKEN}"
: "${LANBOX_PIN:?set LANBOX_PIN}"
for p in ".." "../" "/.." "/docs/../.." "/docs/../../etc/passwd" "%2e%2e/%2e%2e/x" "..%2f.."; do
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "X-PIN: $LANBOX_PIN" "$BASE/api/v1/files?path=$p&token=$LANBOX_TOKEN") || fail "curl $p"
  [ "$code" = "400" ] || fail "$p -> $code (want 400)"
  echo "OK $p -> 400"
done
echo "== PIN gate (needs LANBOX_TOKEN + LANBOX_PIN) =="
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/info?token=$LANBOX_TOKEN") || fail "curl no-pin"
[ "$code" = "401" ] || fail "no PIN -> $code (want 401)"
echo "OK no PIN -> 401"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "X-PIN: 000000" "$BASE/api/v1/info?token=$LANBOX_TOKEN") || fail "curl wrong-pin"
[ "$code" = "401" ] || fail "wrong PIN -> $code (want 401)"
echo "OK wrong PIN -> 401"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "X-PIN: $LANBOX_PIN" "$BASE/api/v1/info?token=$LANBOX_TOKEN") || fail "curl pin"
[ "$code" = "200" ] || fail "right PIN -> $code (want 200)"
echo "OK right PIN -> 200"
echo "ALL GREEN"
