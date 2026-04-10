#!/usr/bin/env bash
# test.sh — run tests for all Go modules
set -euo pipefail

MODULES=(
  api-collector
  api-formatter
  api-collector-go
  api-collector-java
  api-collector-node
  api-collector-python
  api-formatter-curl
  api-formatter-markdown
  api-formatter-postman
  api-master
  apilot-cli
)

FAILED=()

for mod in "${MODULES[@]}"; do
  if [ -f "$mod/go.mod" ]; then
    echo "==> Testing $mod..."
    if ! (cd "$mod" && go test ./...) 2>&1; then
      FAILED+=("$mod")
    fi
  else
    echo "==> Skipping $mod (no go.mod)"
  fi
done

if [ ${#FAILED[@]} -gt 0 ]; then
  echo ""
  echo "FAILED modules:"
  for m in "${FAILED[@]}"; do echo "  - $m"; done
  exit 1
fi

echo ""
echo "All tests passed."