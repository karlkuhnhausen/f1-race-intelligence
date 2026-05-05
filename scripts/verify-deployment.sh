#!/usr/bin/env bash
# Post-deployment verification script for F1 Race Intelligence.
# Validates all API endpoints return correct data shapes and values.
#
# Usage:
#   ./scripts/verify-deployment.sh                    # uses default prod URL
#   BASE_URL=http://localhost:8080 ./scripts/verify-deployment.sh
#   YEAR=2026 ./scripts/verify-deployment.sh          # override season
#
# Exit codes:
#   0 — all checks passed
#   1 — one or more checks failed

set -euo pipefail

BASE_URL="${BASE_URL:-http://f1raceintel.westus3.cloudapp.azure.com}"
YEAR="${YEAR:-2026}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
WARN=0

pass() {
  PASS=$((PASS + 1))
  echo -e "  ${GREEN}✓${NC} $1"
}

fail() {
  FAIL=$((FAIL + 1))
  echo -e "  ${RED}✗${NC} $1"
}

warn() {
  WARN=$((WARN + 1))
  echo -e "  ${YELLOW}⚠${NC} $1"
}

section() {
  echo -e "\n${CYAN}━━━ $1 ━━━${NC}"
}

# ─── Health & Readiness ────────────────────────────────────────────────────────
section "Health & Readiness"

HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/healthz")
if [[ "$HTTP_CODE" == "200" ]]; then
  pass "/healthz → 200"
else
  fail "/healthz → $HTTP_CODE (expected 200)"
fi

HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/readyz")
if [[ "$HTTP_CODE" == "204" || "$HTTP_CODE" == "200" ]]; then
  pass "/readyz → $HTTP_CODE"
else
  fail "/readyz → $HTTP_CODE (expected 200 or 204)"
fi

# ─── Calendar ──────────────────────────────────────────────────────────────────
section "Calendar (year=${YEAR})"

CALENDAR=$(curl -s "${BASE_URL}/api/v1/calendar?year=${YEAR}")
ROUND_COUNT=$(echo "$CALENDAR" | jq '.rounds | length')

if [[ "$ROUND_COUNT" -ge 20 && "$ROUND_COUNT" -le 24 ]]; then
  pass "Round count: $ROUND_COUNT (expected 20-24)"
else
  fail "Round count: $ROUND_COUNT (expected 20-24)"
fi

# Count completed rounds
COMPLETED_ROUNDS=$(echo "$CALENDAR" | jq '[.rounds[] | select(.status == "completed")] | length')
SCHEDULED_ROUNDS=$(echo "$CALENDAR" | jq '[.rounds[] | select(.status == "scheduled")] | length')
pass "Completed rounds: $COMPLETED_ROUNDS | Scheduled: $SCHEDULED_ROUNDS"

# Check podium on completed rounds
COMPLETED_WITH_PODIUM=$(echo "$CALENDAR" | jq '[.rounds[] | select(.status == "completed" and .podium != null and (.podium | length) > 0)] | length')
if [[ "$COMPLETED_WITH_PODIUM" -eq "$COMPLETED_ROUNDS" ]]; then
  pass "All $COMPLETED_ROUNDS completed rounds have podium data"
else
  fail "Only $COMPLETED_WITH_PODIUM of $COMPLETED_ROUNDS completed rounds have podium data"
fi

# Check NO podium on scheduled rounds
SCHEDULED_WITH_PODIUM=$(echo "$CALENDAR" | jq '[.rounds[] | select(.status == "scheduled" and .podium != null and (.podium | length) > 0)] | length')
if [[ "$SCHEDULED_WITH_PODIUM" -eq 0 ]]; then
  pass "No scheduled rounds have podium data (clean)"
else
  fail "$SCHEDULED_WITH_PODIUM scheduled rounds have podium data (should be 0)"
fi

# ─── Round Details ─────────────────────────────────────────────────────────────
section "Round Details (completed rounds)"

COMPLETED_ROUND_NUMBERS=$(echo "$CALENDAR" | jq -r '[.rounds[] | select(.status == "completed") | .round] | .[]')

for ROUND in $COMPLETED_ROUND_NUMBERS; do
  ROUND_DATA=$(curl -s "${BASE_URL}/api/v1/rounds/${ROUND}?year=${YEAR}")
  RACE_NAME=$(echo "$ROUND_DATA" | jq -r '.race_name // "unknown"')
  SESSION_COUNT=$(echo "$ROUND_DATA" | jq '.sessions | length')

  if [[ "$SESSION_COUNT" -ge 3 ]]; then
    pass "Round $ROUND ($RACE_NAME): $SESSION_COUNT sessions"
  else
    fail "Round $ROUND ($RACE_NAME): only $SESSION_COUNT sessions (expected ≥3)"
  fi

  # Check each completed session has results
  SESSIONS_WITH_EMPTY_RESULTS=$(echo "$ROUND_DATA" | jq '[.sessions[] | select(.status == "completed" and (.results | length) == 0)] | length')
  if [[ "$SESSIONS_WITH_EMPTY_RESULTS" -eq 0 ]]; then
    pass "Round $ROUND: all completed sessions have results"
  else
    EMPTY_NAMES=$(echo "$ROUND_DATA" | jq -r '[.sessions[] | select(.status == "completed" and (.results | length) == 0) | .session_name] | join(", ")')
    fail "Round $ROUND: $SESSIONS_WITH_EMPTY_RESULTS sessions have empty results: $EMPTY_NAMES"
  fi

  # Check for duplicates in race results
  RACE_RESULTS=$(echo "$ROUND_DATA" | jq '[.sessions[] | select(.session_type == "race") | .results[].driver_number] | length')
  RACE_UNIQUE=$(echo "$ROUND_DATA" | jq '[.sessions[] | select(.session_type == "race") | .results[].driver_number] | unique | length')
  if [[ "$RACE_RESULTS" -eq "$RACE_UNIQUE" ]]; then
    pass "Round $ROUND race: no duplicate drivers ($RACE_RESULTS results)"
  else
    fail "Round $ROUND race: $RACE_RESULTS results but only $RACE_UNIQUE unique drivers (duplicates!)"
  fi

  # Check result count is reasonable (≤24 per session — accounts for reserve drivers in practice)
  MAX_RESULTS=$(echo "$ROUND_DATA" | jq '[.sessions[].results | length] | max')
  if [[ "$MAX_RESULTS" -le 24 ]]; then
    pass "Round $ROUND: max results per session = $MAX_RESULTS (≤24)"
  else
    fail "Round $ROUND: max results per session = $MAX_RESULTS (>24, likely duplicates)"
  fi
done

# ─── Standings ─────────────────────────────────────────────────────────────────
section "Standings"

DRIVERS=$(curl -s "${BASE_URL}/api/v1/standings/drivers?year=${YEAR}")
DRIVER_COUNT=$(echo "$DRIVERS" | jq '.rows | length')
if [[ "$DRIVER_COUNT" -ge 18 && "$DRIVER_COUNT" -le 22 ]]; then
  pass "Driver standings: $DRIVER_COUNT drivers"
else
  fail "Driver standings: $DRIVER_COUNT drivers (expected 18-22)"
fi

# Check points are descending
POINTS_SORTED=$(echo "$DRIVERS" | jq '[.rows[].points] | . == (. | sort_by(-.) )')
if [[ "$POINTS_SORTED" == "true" ]]; then
  pass "Driver standings: points in descending order"
else
  fail "Driver standings: points NOT in descending order"
fi

# Check team_color populated
EMPTY_COLORS=$(echo "$DRIVERS" | jq '[.rows[] | select(.team_color == "" or .team_color == null)] | length')
if [[ "$EMPTY_COLORS" -eq 0 ]]; then
  pass "Driver standings: all rows have team_color"
else
  warn "Driver standings: $EMPTY_COLORS of $DRIVER_COUNT rows have empty team_color"
fi

CONSTRUCTORS=$(curl -s "${BASE_URL}/api/v1/standings/constructors?year=${YEAR}")
TEAM_COUNT=$(echo "$CONSTRUCTORS" | jq '.rows | length')
if [[ "$TEAM_COUNT" -ge 9 && "$TEAM_COUNT" -le 11 ]]; then
  pass "Constructor standings: $TEAM_COUNT teams"
else
  fail "Constructor standings: $TEAM_COUNT teams (expected 9-11)"
fi

# ─── Progression Charts ────────────────────────────────────────────────────────
section "Progression Charts"

DRIVER_PROG=$(curl -s "${BASE_URL}/api/v1/standings/drivers/progression?season=${YEAR}")
PROG_ROUNDS=$(echo "$DRIVER_PROG" | jq '.rounds | length')

if [[ "$PROG_ROUNDS" -ge "$COMPLETED_ROUNDS" ]]; then
  pass "Driver progression: $PROG_ROUNDS rounds (completed races: $COMPLETED_ROUNDS)"
elif [[ "$PROG_ROUNDS" -ge 1 ]]; then
  fail "Driver progression: only $PROG_ROUNDS rounds but $COMPLETED_ROUNDS races completed"
else
  fail "Driver progression: NO rounds (empty chart)"
fi

CONSTRUCTOR_PROG=$(curl -s "${BASE_URL}/api/v1/standings/constructors/progression?season=${YEAR}")
CPROG_ROUNDS=$(echo "$CONSTRUCTOR_PROG" | jq '.rounds | length')

if [[ "$CPROG_ROUNDS" -ge "$COMPLETED_ROUNDS" ]]; then
  pass "Constructor progression: $CPROG_ROUNDS rounds"
elif [[ "$CPROG_ROUNDS" -ge 1 ]]; then
  fail "Constructor progression: only $CPROG_ROUNDS rounds but $COMPLETED_ROUNDS races completed"
else
  fail "Constructor progression: NO rounds (empty chart)"
fi

# ─── Analysis ──────────────────────────────────────────────────────────────────
section "Analysis (completed rounds)"

for ROUND in $COMPLETED_ROUND_NUMBERS; do
  HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/rounds/${ROUND}/sessions/race/analysis?year=${YEAR}")
  if [[ "$HTTP_CODE" == "200" ]]; then
    pass "Round $ROUND race analysis → 200"
  elif [[ "$HTTP_CODE" == "404" ]]; then
    warn "Round $ROUND race analysis → 404 (data may not be populated yet)"
  else
    fail "Round $ROUND race analysis → $HTTP_CODE"
  fi
done

# ─── Summary ───────────────────────────────────────────────────────────────────
section "Summary"
echo -e "  ${GREEN}Passed: $PASS${NC} | ${RED}Failed: $FAIL${NC} | ${YELLOW}Warnings: $WARN${NC}"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
  echo -e "${RED}DEPLOYMENT VERIFICATION FAILED${NC} — $FAIL check(s) need attention"
  exit 1
else
  echo -e "${GREEN}DEPLOYMENT VERIFICATION PASSED${NC}"
  exit 0
fi
