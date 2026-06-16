#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
DB_PATH="${1:-$PROJECT_ROOT/database/price.db}"
SQL_FILE="$SCRIPT_DIR/sql/mail_account_seed.sql"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 command not found"
  exit 1
fi

if [ ! -f "$SQL_FILE" ]; then
  echo "seed sql not found: $SQL_FILE"
  exit 1
fi

if [ ! -f "$DB_PATH" ]; then
  echo "database not found: $DB_PATH"
  exit 1
fi

sqlite3 "$DB_PATH" < "$SQL_FILE"

add_column_if_missing() {
  local table_name="$1"
  local column_name="$2"
  local column_def="$3"
  if ! sqlite3 "$DB_PATH" "PRAGMA table_info(${table_name});" | cut -d'|' -f2 | grep -qx "$column_name"; then
    sqlite3 "$DB_PATH" "ALTER TABLE ${table_name} ADD COLUMN ${column_name} ${column_def};"
  fi
}

add_column_if_missing "mail_account" "is_current_running" "VARCHAR(10) DEFAULT '0'"
add_column_if_missing "mail_account" "current_run_mark_time" "VARCHAR(255) DEFAULT ''"
add_column_if_missing "mail_account" "proton_registered" "VARCHAR(10) DEFAULT '0'"
add_column_if_missing "mail_account" "proton_email" "VARCHAR(255) DEFAULT ''"
add_column_if_missing "mail_account" "proton_password" "VARCHAR(255) DEFAULT 'Zhangzhi@888'"
add_column_if_missing "mail_account" "codex_device_auth_id" "TEXT DEFAULT ''"
add_column_if_missing "mail_account" "codex_authorization_code" "TEXT DEFAULT ''"
add_column_if_missing "mail_account" "codex_code_verifier" "TEXT DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_json" "TEXT DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_plan_type" "VARCHAR(100) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_allowed" "VARCHAR(20) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_limit_reached" "VARCHAR(20) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_used_percent" "VARCHAR(50) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_reset_at" "VARCHAR(255) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_last_query_time" "VARCHAR(255) DEFAULT ''"
add_column_if_missing "mail_account" "codex_usage_msg" "TEXT DEFAULT ''"

sqlite3 "$DB_PATH" <<'SQL'
UPDATE mail_account
SET is_current_running = ifnull(is_current_running, '0')
WHERE is_current_running IS NULL OR is_current_running = '';

UPDATE mail_account
SET proton_registered = ifnull(proton_registered, '0')
WHERE proton_registered IS NULL OR proton_registered = '';

UPDATE mail_account
SET proton_email = CASE
  WHEN instr(email_name, '@') > 1 THEN substr(email_name, 1, instr(email_name, '@') - 1) || '@proton.me'
  ELSE ''
END
WHERE ifnull(proton_email, '') = '';

UPDATE mail_account
SET proton_password = 'Zhangzhi@888'
WHERE ifnull(proton_password, '') = '';
SQL

echo "seed applied: $DB_PATH"
