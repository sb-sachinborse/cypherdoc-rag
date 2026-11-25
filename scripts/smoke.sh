#!/bin/sh
# End-to-end smoke test against a running stack:  make smoke
set -eu
BASE=${BASE:-http://localhost:8080}
JAR=$(mktemp); JAR2=$(mktemp); TMP=$(mktemp)
trap 'rm -f "$JAR" "$JAR2" "$TMP" "$TMP.txt" "$TMP.exe"' EXIT
H='X-Requested-With: cypherdoc'
EMAIL="smoke$$-$(date +%s)@example.com"; EMAIL2="other$$-$(date +%s)@example.com"; PW="smoke-test-password-1"
pass() { echo "  ok   $1"; }
fail() { echo "  FAIL $1"; exit 1; }
code() { curl -s -o "$TMP" -w '%{http_code}' "$@"; }

echo "Waiting for the app and the AI models (first run downloads ~2 GB)..."
i=0
until curl -fs "$BASE/api/status" | grep -q '"ready":true'; do
  i=$((i+1)); [ $i -gt 360 ] && fail "models not ready after 30 minutes"; sleep 5
done
pass "models ready"

[ "$(code "$BASE/api/documents")" = 401 ] && pass "unauthenticated request rejected (401)" || fail "expected 401"
[ "$(code -X POST -H 'Content-Type: application/json' -d '{}' "$BASE/api/auth/login")" = 403 ] && pass "missing CSRF header rejected (403)" || fail "expected 403"
[ "$(code -H "$H" -H 'Content-Type: application/json' -d '{"email":"a@b.co","password":"short"}' "$BASE/api/auth/register")" = 400 ] && pass "weak password rejected (400)" || fail "expected 400"

[ "$(code -c "$JAR" -H "$H" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\",\"name\":\"Smoke\"}" "$BASE/api/auth/register")" = 201 ] && pass "register" || fail "register"
[ "$(code -b "$JAR" "$BASE/api/auth/me")" = 200 ] && pass "session works" || fail "me"

printf 'Acme Corp leave policy. Full-time employees receive 24 days of paid leave per year. Remote work is allowed three days per week.' > "$TMP.txt"
[ "$(code -b "$JAR" -H "$H" -F collection=Smoke -F "file=@$TMP.txt;filename=hr.txt" "$BASE/api/documents")" = 202 ] && pass "upload accepted" || fail "upload"
printf 'MZ not really text\000binary' > "$TMP.exe"
[ "$(code -b "$JAR" -H "$H" -F "file=@$TMP.exe;filename=evil.pdf" "$BASE/api/documents")" = 415 ] && pass "fake PDF rejected (415)" || fail "expected 415"

i=0
until curl -fs -b "$JAR" "$BASE/api/documents" | grep -q '"status":"indexed"'; do
  i=$((i+1)); [ $i -gt 120 ] && fail "document not indexed in time"; sleep 3
done
pass "document indexed"

OUT=$(curl -s -N -m 300 -b "$JAR" -H "$H" -H 'Content-Type: application/json' -d '{"question":"How many days of paid leave do employees get?","collection":""}' "$BASE/api/chat")
echo "$OUT" | grep -q 'event: sources' && echo "$OUT" | grep -q 'hr.txt' && pass "answer cites hr.txt" || fail "no citation in: $OUT"
echo "$OUT" | grep -q '24' && pass "answer mentions 24 days" || echo "  warn answer did not mention 24 (small model); check manually"

code -c "$JAR2" -H "$H" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL2\",\"password\":\"$PW\"}" "$BASE/api/auth/register" >/dev/null
[ "$(curl -s -b "$JAR2" "$BASE/api/documents")" = "[]" ] && pass "other user sees no documents" || fail "data leak between users"
OUT2=$(curl -s -N -m 120 -b "$JAR2" -H "$H" -H 'Content-Type: application/json' -d '{"question":"How many days of paid leave do employees get?"}' "$BASE/api/chat")
echo "$OUT2" | grep -q 'hr.txt' && fail "other user retrieved someone else's document" || pass "other user cannot retrieve it"

echo "All checks passed."
