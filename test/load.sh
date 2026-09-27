#!/usr/bin/env bash
# M3 load: N parallel uploads; counts 201 vs 429 (429 is correct behavior
# past 4 slots, not a failure). Usage: ./test/load.sh [base] [token] [N]
set -u
BASE="${1:-http://localhost:8080}"
TOK="${2:?set token}"
N="${3:-4}"
dd if=/dev/zero of=/tmp/load-20M bs=1M count=20 status=none
dir=$(mktemp -d)
for i in $(seq 1 "$N"); do
  (curl -s -o /dev/null -w "%{http_code}" \
    -F "file=@/tmp/load-20M;filename=load$i" \
    "$BASE/api/v1/files/upload?path=/&token=$TOK" > "$dir/$i") &
done
wait
ok=$(grep -l 201 "$dir"/* 2>/dev/null | wc -l)
busy=$(grep -l 429 "$dir"/* 2>/dev/null | wc -l)
other=$((N - ok - busy))
echo "clients=$N ok_201=$ok busy_429=$busy other=$other"
rm -rf "$dir" /tmp/load-20M
[ "$other" = "0" ]
