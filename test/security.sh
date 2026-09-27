#!/usr/bin/env bash
# M2 security sweeps: no-token -> 401 everywhere, traversal -> 400.
# Usage: LANBOX_TOKEN=<token> ./test/security.sh [base]
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

echo "== traversal sweep (want 400, needs LANBOX_TOKEN) =="
: "${LANBOX_TOKEN:?set LANBOX_TOKEN}"
for p in ".." "../" "/.." "/docs/../.." "/docs/../../etc/passwd" "%2e%2e/%2e%2e/x" "..%2f.."; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/files?path=$p&token=$LANBOX_TOKEN") || fail "curl $p"
  [ "$code" = "400" ] || fail "$p -> $code (want 400)"
  echo "OK $p -> 400"
done
echo "ALL GREEN"
