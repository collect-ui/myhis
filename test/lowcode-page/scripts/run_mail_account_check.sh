#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
RESULT_ROOT="$PROJECT_ROOT/test/lowcode-page/results"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LATEST_DIR="$RESULT_ROOT/latest"
HISTORY_DIR="$RESULT_ROOT/history/mail_account/$TIMESTAMP"

mkdir -p "$LATEST_DIR" "$HISTORY_DIR"
rm -f "$LATEST_DIR"/* 2>/dev/null || true

export MAIL_ACCOUNT_OUTPUT_DIR="$LATEST_DIR"
FRONTEND_STATUS="- [ ] frontend page check depends on node + playwright availability"

"$SCRIPT_DIR/apply_mail_account_seed.sh"
"$SCRIPT_DIR/backend/mail_account_api_check.sh"

if command -v node >/dev/null 2>&1; then
  if "$SCRIPT_DIR/frontend/mail_account_page_check.js"; then
    FRONTEND_STATUS="- [x] frontend page check executed"
  else
    FRONTEND_STATUS="- [ ] frontend page check failed or playwright unavailable"
    echo "frontend page check failed or playwright unavailable" | tee -a "$LATEST_DIR/frontend.log"
  fi
else
  FRONTEND_STATUS="- [ ] frontend page check skipped: node command not found"
  echo "node command not found, skip frontend page check" | tee -a "$LATEST_DIR/frontend.log"
fi

cp -a "$LATEST_DIR"/. "$HISTORY_DIR"/

cat > "$LATEST_DIR/checklist.md" <<EOF
# Mail Account Checklist

- [x] seed sql applied
- [x] backend api check executed
$FRONTEND_STATUS
- [x] latest results copied to history
EOF

echo "mail_account check completed: $HISTORY_DIR"
