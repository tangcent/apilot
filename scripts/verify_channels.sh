#!/bin/bash
# verify_channels.sh — End-to-end verification that apilot-cli can export APIs
# from each sample project into every registered output channel.
#
# Channels verified:
#   - markdown  (writes a .md file)
#   - curl      (writes a .sh file)
#   - postman   (pushes a real collection to the Postman API)
#
# Usage:
#   PILOT_POSTMAN_API_KEY=<your-key> scripts/verify_channels.sh
#
# If PILOT_POSTMAN_API_KEY is unset, the postman channel is skipped (reported
# as SKIP in the matrix) and the run exits non-zero so CI catches it.
#
# The script locates itself via $(dirname "$0") and resolves the repo root
# relative to that, so it can be invoked from any working directory.
#
# Compatible with bash 3.2 (macOS default) — no associative arrays.

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SAMPLES_DIR="$REPO_ROOT/samples"
VERIFY_DIR="$REPO_ROOT/.verify"
BIN_DIR="$REPO_ROOT/bin"
PILOT_BIN="$BIN_DIR/apilot"

SAMPLES=(go-gin java-springmvc node-express python-fastapi)
CHANNELS=(markdown curl postman)

# PASS/FAIL matrix stored as a flat file: "<sample> <channel> <status>"
MATRIX_FILE="$VERIFY_DIR/.matrix.tmp"
: > "$MATRIX_FILE"

# Track created Postman collection UIDs for cleanup: "<sample> <uid>"
CLEANUP_FILE="$VERIFY_DIR/.cleanup.tmp"
: > "$CLEANUP_FILE"

OVERALL_STATUS=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

err() { printf '\033[31m[ERR]\033[0m %s\n' "$*" >&2; }
info() { printf '\033[36m[INFO]\033[0m %s\n' "$*"; }
ok() { printf '\033[32m[PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[FAIL]\033[0m %s\n' "$*"; }
skip() { printf '\033[33m[SKIP]\033[0m %s\n' "$*"; }

# Record a matrix result.
record() { echo "$1 $2 $3" >> "$MATRIX_FILE"; }

# Extract a top-level string field from a JSON file.
# Usage: json_get_str <file> <field>
# Uses python3 if available, otherwise falls back to grep/sed.
json_get_str() {
  local file="$1"
  local field="$2"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import json, sys
try:
    with open('$file') as f:
        data = json.load(f)
    val = data.get('$field', '')
    print(val if isinstance(val, str) else '')
except Exception:
    print('')
"
  else
    grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$file" \
      | head -n1 \
      | sed -E 's/.*"'"$field"'"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/'
  fi
}

# ---------------------------------------------------------------------------
# Step 1: Build apilot-cli
# ---------------------------------------------------------------------------

build_apilot() {
  info "Building apilot-cli to $PILOT_BIN ..."
  (cd "$REPO_ROOT" && go build -o "$PILOT_BIN" ./apilot-cli) || {
    err "build failed"
    return 1
  }
  if ! "$PILOT_BIN" --version >/dev/null 2>&1; then
    err "apilot --version failed"
    return 1
  fi
  info "apilot version: $("$PILOT_BIN" --version)"
  return 0
}

# ---------------------------------------------------------------------------
# Step 2: Markdown channel
# ---------------------------------------------------------------------------

verify_markdown() {
  local sample="$1"
  local out_dir="$VERIFY_DIR/$sample/markdown"
  local out_file="$out_dir/api.md"
  mkdir -p "$out_dir"

  if ! "$PILOT_BIN" "$SAMPLES_DIR/$sample" \
        --formatter markdown \
        --output "$out_file" >/dev/null 2>"$out_dir/stderr.log"; then
    err "$sample/markdown: apilot export failed (see $out_dir/stderr.log)"
    return 1
  fi

  if [ ! -f "$out_file" ]; then
    err "$sample/markdown: output file not created"
    return 1
  fi

  local pass=1
  grep -q "GET /users" "$out_file" || { err "$sample/markdown: missing 'GET /users'"; pass=0; }
  grep -q "POST /users" "$out_file" || { err "$sample/markdown: missing 'POST /users'"; pass=0; }
  # Accept any path-parameter name: {id}, {user_id}, :id, :user_id
  grep -qE "GET /users/(\{[a-zA-Z_][a-zA-Z0-9_]*\}|:[a-zA-Z_][a-zA-Z0-9_]*)" "$out_file" \
    || { err "$sample/markdown: missing 'GET /users/{param}'"; pass=0; }

  [ "$pass" = "1" ]
}

# ---------------------------------------------------------------------------
# Step 3: cURL channel
# ---------------------------------------------------------------------------

verify_curl() {
  local sample="$1"
  local out_dir="$VERIFY_DIR/$sample/curl"
  local out_file="$out_dir/api.sh"
  mkdir -p "$out_dir"

  if ! "$PILOT_BIN" "$SAMPLES_DIR/$sample" \
        --formatter curl \
        --output "$out_file" >/dev/null 2>"$out_dir/stderr.log"; then
    err "$sample/curl: apilot export failed (see $out_dir/stderr.log)"
    return 1
  fi

  if [ ! -f "$out_file" ]; then
    err "$sample/curl: output file not created"
    return 1
  fi

  local pass=1
  grep -q "curl -X GET" "$out_file" || { err "$sample/curl: missing 'curl -X GET'"; pass=0; }
  grep -q "curl -X POST" "$out_file" || { err "$sample/curl: missing 'curl -X POST'"; pass=0; }
  grep -qE "/users(/|\?|$)" "$out_file" || { err "$sample/curl: missing '/users' path"; pass=0; }

  [ "$pass" = "1" ]
}

# ---------------------------------------------------------------------------
# Step 4: Postman channel (real API push)
# ---------------------------------------------------------------------------

verify_postman() {
  local sample="$1"
  local out_dir="$VERIFY_DIR/$sample/postman"
  local result_file="$out_dir/result.json"
  local fetched_file="$out_dir/fetched.json"
  mkdir -p "$out_dir"

  if [ -z "${PILOT_POSTMAN_API_KEY:-}" ]; then
    skip "$sample/postman: PILOT_POSTMAN_API_KEY not set"
    return 2  # 2 means SKIP
  fi

  local collection_name="apilot-verify-$sample"
  local params
  params="{\"collectionName\":\"$collection_name\",\"baseURL\":\"http://localhost:8080\"}"

  info "$sample/postman: pushing collection '$collection_name' to Postman API ..."
  if ! "$PILOT_BIN" "$SAMPLES_DIR/$sample" \
        --formatter postman \
        --params "$params" \
        > "$result_file" 2>"$out_dir/stderr.log"; then
    err "$sample/postman: apilot push failed (see $out_dir/stderr.log)"
    cat "$result_file" >&2 2>/dev/null || true
    return 1
  fi

  local uid action
  uid="$(json_get_str "$result_file" collectionUid)"
  action="$(json_get_str "$result_file" action)"

  if [ -z "$uid" ]; then
    err "$sample/postman: collectionUid empty in result.json"
    cat "$result_file" >&2 2>/dev/null || true
    return 1
  fi
  if [ "$action" != "created" ]; then
    err "$sample/postman: expected action='created', got '$action'"
    return 1
  fi

  info "$sample/postman: created collection uid=$uid — fetching back for verification ..."
  echo "$sample $uid" >> "$CLEANUP_FILE"

  if ! curl -sf -H "X-Api-Key: $PILOT_POSTMAN_API_KEY" \
        "https://api.getpostman.com/collections/$uid" \
        > "$fetched_file" 2>"$out_dir/fetch_err.log"; then
    err "$sample/postman: GET collection failed (see $out_dir/fetch_err.log)"
    return 1
  fi

  local pass=1
  grep -q "\"$collection_name\"" "$fetched_file" \
    || { err "$sample/postman: collection name '$collection_name' not found in fetched.json"; pass=0; }
  grep -q "/users" "$fetched_file" \
    || { err "$sample/postman: '/users' path not found in fetched.json"; pass=0; }
  grep -q "\"GET\"" "$fetched_file" \
    || { err "$sample/postman: method GET not found in fetched.json"; pass=0; }
  grep -q "\"POST\"" "$fetched_file" \
    || { err "$sample/postman: method POST not found in fetched.json"; pass=0; }

  [ "$pass" = "1" ]
}

# ---------------------------------------------------------------------------
# Step 5: Postman cleanup (best-effort)
# ---------------------------------------------------------------------------

cleanup_postman() {
  local sample="$1"
  local uid="$2"
  if [ -z "$uid" ]; then
    return 0
  fi
  info "$sample/postman: deleting collection uid=$uid (best-effort) ..."
  local resp
  resp="$(curl -s -X DELETE -H "X-Api-Key: $PILOT_POSTMAN_API_KEY" \
        "https://api.getpostman.com/collections/$uid" 2>/dev/null || true)"
  if echo "$resp" | grep -q "collection" 2>/dev/null; then
    info "$sample/postman: cleanup ok"
  else
    info "$sample/postman: cleanup returned non-matching response (ignored)"
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  info "Repo root: $REPO_ROOT"
  info "Verify dir: $VERIFY_DIR"
  mkdir -p "$VERIFY_DIR"

  if ! build_apilot; then
    err "aborting: build failed"
    exit 1
  fi

  # One-time Postman API key setup (only if key is provided).
  if [ -n "${PILOT_POSTMAN_API_KEY:-}" ]; then
    info "Setting postman.api.key in apilot settings store ..."
    if ! "$PILOT_BIN" set postman.api.key "$PILOT_POSTMAN_API_KEY" >/dev/null 2>&1; then
      err "failed to set postman.api.key"
      exit 1
    fi
  else
    skip "PILOT_POSTMAN_API_KEY not set — postman channel will be SKIP for all samples"
  fi

  for sample in "${SAMPLES[@]}"; do
    # --- markdown ---
    if verify_markdown "$sample"; then
      ok "$sample/markdown"
      record "$sample" "markdown" "PASS"
    else
      fail "$sample/markdown"
      record "$sample" "markdown" "FAIL"
      OVERALL_STATUS=1
    fi

    # --- curl ---
    if verify_curl "$sample"; then
      ok "$sample/curl"
      record "$sample" "curl" "PASS"
    else
      fail "$sample/curl"
      record "$sample" "curl" "FAIL"
      OVERALL_STATUS=1
    fi

    # --- postman ---
    verify_postman "$sample"
    rc=$?
    case "$rc" in
      0)
        ok "$sample/postman"
        record "$sample" "postman" "PASS"
        ;;
      2)
        record "$sample" "postman" "SKIP"
        OVERALL_STATUS=1
        ;;
      *)
        fail "$sample/postman"
        record "$sample" "postman" "FAIL"
        OVERALL_STATUS=1
        ;;
    esac
  done

  # Best-effort cleanup of created Postman collections.
  if [ -n "${PILOT_POSTMAN_API_KEY:-}" ] && [ -s "$CLEANUP_FILE" ]; then
    info "=== Postman cleanup (best-effort) ==="
    while read -r csample cuid; do
      [ -n "$csample" ] || continue
      cleanup_postman "$csample" "$cuid"
    done < "$CLEANUP_FILE"
  fi

  # Print matrix.
  echo
  echo "==================== PASS/FAIL MATRIX ===================="
  printf '%-20s' "sample"
  for ch in "${CHANNELS[@]}"; do
    printf '%-12s' "$ch"
  done
  printf '\n'
  for sample in "${SAMPLES[@]}"; do
    printf '%-20s' "$sample"
    for ch in "${CHANNELS[@]}"; do
      cell="$(awk -v s="$sample" -v c="$ch" '$1==s && $2==c {print $3}' "$MATRIX_FILE")"
      printf '%-12s' "${cell:-?}"
    done
    printf '\n'
  done
  echo "=========================================================="

  rm -f "$MATRIX_FILE" "$CLEANUP_FILE"

  if [ "$OVERALL_STATUS" = "0" ]; then
    ok "All cells PASS"
  else
    err "One or more cells FAILED or were SKIPped"
  fi
  exit "$OVERALL_STATUS"
}

main "$@"
