#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${MAIL_ACCOUNT_BASE_URL:-http://192.168.232.130:8015}"
API_URL="${MAIL_ACCOUNT_API_URL:-$BASE_URL/template_data/data}"
USERNAME="${MAIL_ACCOUNT_USERNAME:-admin}"
PASSWORD="${MAIL_ACCOUNT_PASSWORD:-123456}"
OUTPUT_DIR="${MAIL_ACCOUNT_OUTPUT_DIR:-$(pwd)}"
COOKIE_JAR="$(mktemp)"
TS="$(date +%Y%m%d%H%M%S)"
PREFIX="mail-batch-$TS"

mkdir -p "$OUTPUT_DIR"
BACKEND_LOG="$OUTPUT_DIR/backend.log"
API_RESPONSE="$OUTPUT_DIR/api-response.json"

cleanup() {
  rm -f "$COOKIE_JAR"
}
trap cleanup EXIT

exec > >(tee "$BACKEND_LOG") 2>&1

login_response="$(curl -sS -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.login" \
  -d "{\"service\":\"system.login\",\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")"

echo "$login_response" | jq '.'

login_success="$(echo "$login_response" | jq -r '.success // false')"
if [ "$login_success" != "true" ]; then
  echo "$login_response" > "$API_RESPONSE"
  echo "login failed, set MAIL_ACCOUNT_USERNAME / MAIL_ACCOUNT_PASSWORD"
  exit 1
fi

import_payload="$(cat <<EOF
{
  "service": "system.mail_account_import_batch",
  "email_name_list_sql": "'${PREFIX}-1@example.com','${PREFIX}-2@example.com'",
  "mail_account_list": [
    {
      "order_index": 1,
      "email_name": "${PREFIX}-1@example.com",
      "password": "pwd-${TS}-1",
      "guid_code": "guid-${TS}-1",
      "proton_registered": "0",
      "proton_email": "${PREFIX}-1@proton.me",
      "proton_password": "Zhangzhi@888",
      "recovery_code": "recovery-${TS}-1",
      "raw_text": "${PREFIX}-1@example.com----pwd-${TS}-1----guid-${TS}-1----recovery-${TS}-1"
    },
    {
      "order_index": 2,
      "email_name": "${PREFIX}-2@example.com",
      "password": "pwd-${TS}-2",
      "guid_code": "guid-${TS}-2",
      "proton_registered": "0",
      "proton_email": "${PREFIX}-2@proton.me",
      "proton_password": "Zhangzhi@888",
      "recovery_code": "recovery-${TS}-2",
      "raw_text": "${PREFIX}-2@example.com----pwd-${TS}-2----guid-${TS}-2----recovery-${TS}-2"
    }
  ]
}
EOF
)"

first_import="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_import_batch" \
  -d "$import_payload")"

echo "$first_import" | jq '.'
echo "$first_import" > "$API_RESPONSE"

first_create_count="$(echo "$first_import" | jq -r '.data.create_count // 0')"
if [ "$first_create_count" != "2" ]; then
  echo "expected first import create_count=2, got $first_create_count"
  exit 1
fi

second_import="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_import_batch" \
  -d "$import_payload")"

echo "$second_import" | jq '.'

second_skip_count="$(echo "$second_import" | jq -r '.data.skip_count // 0')"
if [ "$second_skip_count" != "2" ]; then
  echo "expected second import skip_count=2, got $second_skip_count"
  exit 1
fi

query_response="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_query" \
  -d "{\"service\":\"system.mail_account_query\",\"search\":\"$PREFIX\",\"page\":1,\"size\":20}")"

echo "$query_response" | jq '.'

query_count="$(echo "$query_response" | jq -r '.count // 0')"
if [ "$query_count" -lt 2 ]; then
  echo "expected query count >= 2, got $query_count"
  exit 1
fi

first_mail_account_id="$(echo "$query_response" | jq -r '.data[0].mail_account_id // empty')"
settings_response="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_save_settings" \
  -d "{\"service\":\"system.mail_account_save_settings\",\"mail_account_id\":\"$first_mail_account_id\",\"is_current_running\":\"1\",\"proton_registered\":\"1\",\"proton_email\":\"${PREFIX}-1@proton.me\",\"proton_password\":\"Zhangzhi@888\"}")"

echo "$settings_response" | jq '.'

settings_verify="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_query" \
  -d "{\"service\":\"system.mail_account_query\",\"mail_account_id\":\"$first_mail_account_id\",\"page\":1,\"size\":1}")"

echo "$settings_verify" | jq '.'

saved_current_flag="$(echo "$settings_verify" | jq -r '.data[0].is_current_running // "0"')"
saved_proton_flag="$(echo "$settings_verify" | jq -r '.data[0].proton_registered // "0"')"
if [ "$saved_current_flag" != "1" ] || [ "$saved_proton_flag" != "1" ]; then
  echo "expected settings save to persist current/proton flags"
  exit 1
fi

delete_ids="$(echo "$query_response" | jq -c '[.data[].mail_account_id]')"
delete_response="$(curl -sS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "$API_URL?service=system.mail_account_delete" \
  -d "{\"service\":\"system.mail_account_delete\",\"mail_account_id_list\":$delete_ids}")"

echo "$delete_response" | jq '.'

summary_file="$OUTPUT_DIR/summary.md"
cat > "$summary_file" <<EOF
# Mail Account API Check

- Base URL: $BASE_URL
- Imported prefix: $PREFIX
- First import create_count: $first_create_count
- Second import skip_count: $second_skip_count
- Query count after import: $query_count
- Saved current flag: $saved_current_flag
- Saved proton flag: $saved_proton_flag
- Cleanup delete ids: $delete_ids
EOF

echo "backend api check completed"
