#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SPORT_WINDOW_LOCAL_DIR:-$(cd "${SCRIPT_DIR}/../../../.." && pwd)}"

HOST="${SPORT_WINDOW_HOST:-10.96.32.180}"
USER_NAME="${SPORT_WINDOW_USER:-18874948657}"
PASSWORD="${SPORT_WINDOW_SSH_PASSWORD:-${WINDOW_SERVER_PASSWORD:-}}"
REMOTE_DIR_PS="${SPORT_WINDOW_REMOTE_DIR:-F:\moon-windows-amd64-clean}"
REMOTE_DIR_SCP="${REMOTE_DIR_PS//\\//}"
REMOTE_SYNC_DIR_SCP="${REMOTE_DIR_SCP}/_sync"
REMOTE_POWERSHELL="${SPORT_WINDOW_POWERSHELL:-powershell}"
REMOTE_SQLITE_DIR_PS="${SPORT_WINDOW_REMOTE_SQLITE_DIR:-${REMOTE_DIR_PS}\\tools\\sqlite}"
REMOTE_SQLITE_EXE_PS="${SPORT_WINDOW_REMOTE_SQLITE3:-${REMOTE_SQLITE_DIR_PS}\\sqlite3.exe}"
APP_PORT="${SPORT_WINDOW_APP_PORT:-8009}"

BUILD_EXE="${SPORT_WINDOW_BUILD_EXE:-windows/main.exe}"
PAYLOAD_NAME="sport-window-sync-payload.tgz"
SQLITE_TOOLS_URL="${SPORT_WINDOW_SQLITE_TOOLS_URL:-https://www.sqlite.org/2026/sqlite-tools-win-x64-3530100.zip}"
SQLITE_TOOLS_SHA3="${SPORT_WINDOW_SQLITE_TOOLS_SHA3:-14dc1653482c75dd83e3932b378ff4e8249d3774f4098046ac500eb8452c5ccf}"
SQLITE_TOOLS_ARCHIVE="${SQLITE_TOOLS_URL##*/}"

DRY_RUN=0
CHECK_ONLY=0
VERIFY_ONLY=0
INCLUDE_EXE=0
SKIP_BUILD=0
SKIP_DB_CHECK=0
SKIP_CONFIG_CHECK=0
ASSUME_CONTINUE_WITHOUT_DB=0
ALLOW_FILE_ONLY_WITH_GO_CHANGES=0
INSTALL_REMOTE_SQLITE=0

SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o LogLevel=ERROR
)

TMP_DIR=""

usage() {
  cat <<'EOF'
Usage:
  bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh [options]

Options:
  --dry-run                      Check/package plan only; do not write remote files.
  --check-only                   Only check remote DB DDL and config keys.
  --verify-only                  Only compare local/remote deployed file hashes.
  --include-exe                  Also build/sync/verify windows/main.exe. Default is frontend+collect only.
  --install-remote-sqlite        Install/upgrade Windows sqlite3 under remote tools\sqlite, then exit.
                                 Used only for read-only DDL comparison; never transfers database files.
  --skip-build                   Reuse windows/main.exe.
  --skip-db-check                Do not compare database/*.db schema.
  --skip-config-check            Do not compare conf/application.properties keys.
  --continue-without-db-sync     Continue deploy when DDL differs, without changing DB.
                                 This script never backs up, uploads, or overwrites database/*.db.
  --allow-file-only-with-go-changes
                                 Continue frontend/collect-only sync even when Go/model/plugin
                                 changes suggest main.exe should also be rebuilt.
  -h, --help                     Show help.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --check-only)
      CHECK_ONLY=1
      shift
      ;;
    --verify-only)
      VERIFY_ONLY=1
      SKIP_DB_CHECK=1
      SKIP_CONFIG_CHECK=1
      shift
      ;;
    --include-exe)
      INCLUDE_EXE=1
      shift
      ;;
    --install-remote-sqlite)
      INSTALL_REMOTE_SQLITE=1
      shift
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    --skip-db-check)
      SKIP_DB_CHECK=1
      shift
      ;;
    --skip-config-check)
      SKIP_CONFIG_CHECK=1
      shift
      ;;
    --continue-without-db-sync)
      ASSUME_CONTINUE_WITHOUT_DB=1
      shift
      ;;
    --allow-file-only-with-go-changes)
      ALLOW_FILE_ONLY_WITH_GO_CHANGES=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "${PASSWORD}" ]]; then
  echo "Missing SSH password. Set SPORT_WINDOW_SSH_PASSWORD or WINDOW_SERVER_PASSWORD." >&2
  exit 2
fi

require_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Required command not found: ${cmd}" >&2
    exit 2
  fi
}

cleanup() {
  if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

confirm() {
  local prompt="$1"
  local answer
  if [[ ! -t 0 ]]; then
    echo "${prompt} [y/N] skipped: stdin is not interactive." >&2
    return 1
  fi
  read -r -p "${prompt} [y/N] " answer
  case "${answer}" in
    y|Y|yes|YES|Yes|是)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

run_remote_ps() {
  local ps_script="$1"
  local encoded
  encoded="$(printf '%s' "\$ProgressPreference='SilentlyContinue'; ${ps_script}" | iconv -f UTF-8 -t UTF-16LE | base64 -w0)"
  SSHPASS="${PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${USER_NAME}@${HOST}" \
    "${REMOTE_POWERSHELL} -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand ${encoded}"
}

scp_to_remote() {
  local local_path="$1"
  local remote_path="$2"
  SSHPASS="${PASSWORD}" sshpass -e scp "${SSH_OPTS[@]}" "${local_path}" "${USER_NAME}@${HOST}:${remote_path}"
}

scp_from_remote() {
  local remote_path="$1"
  local local_path="$2"
  SSHPASS="${PASSWORD}" sshpass -e scp "${SSH_OPTS[@]}" "${USER_NAME}@${HOST}:${remote_path}" "${local_path}"
}

remote_literal() {
  printf "%s" "$1" | sed "s/'/''/g"
}

ensure_remote_dirs() {
  local target
  target="$(remote_literal "${REMOTE_DIR_PS}")"
  run_remote_ps "\$target='${target}'; New-Item -ItemType Directory -Force -Path \$target | Out-Null; New-Item -ItemType Directory -Force -Path (Join-Path \$target '_sync') | Out-Null"
}

remote_path_exists() {
  local path="$1"
  local escaped
  escaped="$(remote_literal "${path}")"
  run_remote_ps "if (Test-Path -LiteralPath '${escaped}') { exit 0 } else { exit 1 }" >/dev/null 2>&1
}

remote_sqlite_command_ps() {
  local preferred
  preferred="$(remote_literal "${REMOTE_SQLITE_EXE_PS}")"
  cat <<EOF
\$preferred='${preferred}';
if (Test-Path -LiteralPath \$preferred) {
  \$sqlite=\$preferred;
} else {
  \$cmd=Get-Command sqlite3 -ErrorAction SilentlyContinue;
  if (!\$cmd) { exit 1 }
  \$sqlite=\$cmd.Source;
}
EOF
}

build_windows_exe() {
  if [[ "${INCLUDE_EXE}" -eq 0 ]]; then
    echo "Skipping Windows build: --include-exe was not set."
    return
  fi

  if [[ "${SKIP_BUILD}" -eq 1 ]]; then
    echo "Skipping Windows build."
  else
    echo "Building Windows executable: ${BUILD_EXE}"
    (
      cd "${REPO_ROOT}"
      GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "${BUILD_EXE}" main.go
    )
  fi

  if [[ ! -f "${REPO_ROOT}/${BUILD_EXE}" ]]; then
    echo "Windows executable not found: ${REPO_ROOT}/${BUILD_EXE}" >&2
    exit 1
  fi
}

install_remote_sqlite3() {
  local archive="${TMP_DIR}/${SQLITE_TOOLS_ARCHIVE}"
  local actual_hash install_dir sync_dir archive_name
  install_dir="$(remote_literal "${REMOTE_SQLITE_DIR_PS}")"
  sync_dir="$(remote_literal "${REMOTE_DIR_PS}")"
  archive_name="$(remote_literal "${SQLITE_TOOLS_ARCHIVE}")"

  echo "Downloading SQLite tools: ${SQLITE_TOOLS_URL}"
  curl -fL --retry 3 --connect-timeout 10 -o "${archive}" "${SQLITE_TOOLS_URL}"
  actual_hash="$(openssl dgst -sha3-256 -r "${archive}" | awk '{print tolower($1)}')"
  if [[ "${actual_hash}" != "${SQLITE_TOOLS_SHA3}" ]]; then
    echo "SQLite tools SHA3 mismatch: expected=${SQLITE_TOOLS_SHA3} actual=${actual_hash}" >&2
    exit 1
  fi

  ensure_remote_dirs
  echo "Uploading SQLite tools to ${USER_NAME}@${HOST}:${REMOTE_SYNC_DIR_SCP}/${SQLITE_TOOLS_ARCHIVE}"
  scp_to_remote "${archive}" "${REMOTE_SYNC_DIR_SCP}/${SQLITE_TOOLS_ARCHIVE}" >/dev/null

  run_remote_ps "\
\$ErrorActionPreference='Stop';
\$target='${sync_dir}';
\$sync=Join-Path \$target '_sync';
\$archive=Join-Path \$sync '${archive_name}';
\$extract=Join-Path \$sync 'sqlite-tools-extract';
\$install='${install_dir}';
if (Test-Path -LiteralPath \$extract) { Remove-Item -Recurse -Force -LiteralPath \$extract };
New-Item -ItemType Directory -Force -Path \$extract | Out-Null;
New-Item -ItemType Directory -Force -Path \$install | Out-Null;
Expand-Archive -LiteralPath \$archive -DestinationPath \$extract -Force;
\$tools=Get-ChildItem -LiteralPath \$extract -Recurse -File -Include sqlite3.exe,sqldiff.exe,sqlite3_analyzer.exe,sqlite3_rsync.exe;
if (!(\$tools | Where-Object { \$_.Name -ieq 'sqlite3.exe' })) { throw 'sqlite3.exe not found in archive' };
foreach (\$tool in \$tools) { Copy-Item -Force -LiteralPath \$tool.FullName -Destination (Join-Path \$install \$tool.Name) };
\$sqlite=Join-Path \$install 'sqlite3.exe';
Write-Output ('SQLite installed: ' + \$sqlite);
& \$sqlite -version
"
}

binary_relevant_changes() {
  if ! git -C "${REPO_ROOT}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    return 0
  fi

  git -C "${REPO_ROOT}" status --porcelain -- \
    main.go \
    go.mod \
    go.sum \
    model \
    plugins
}

check_binary_rebuild_needed() {
  if [[ "${INCLUDE_EXE}" -eq 1 || "${ALLOW_FILE_ONLY_WITH_GO_CHANGES}" -eq 1 ]]; then
    return
  fi

  local changes
  changes="$(binary_relevant_changes)"
  if [[ -z "${changes}" ]]; then
    return
  fi

  echo "Binary-relevant local changes detected:"
  printf '%s\n' "${changes}" | sed -n '1,80p' | sed 's/^/  /'
  echo "New model fields/tables, plugin handlers, or startup wiring usually require --include-exe."

  if [[ "${DRY_RUN}" -eq 1 || "${CHECK_ONLY}" -eq 1 ]]; then
    echo "Dry/check mode: no files were changed, but default sync would need explicit operator choice."
    return
  fi

  if confirm "Continue syncing only frontend/collect without rebuilding or replacing main.exe?"; then
    return
  fi

  echo "Stopped before file sync. Re-run with --include-exe after approval, or use --allow-file-only-with-go-changes for a deliberate file-only sync."
  exit 1
}

properties_keys() {
  local file="$1"
  awk '
    /^[[:space:]]*($|#|!)/ { next }
    {
      line=$0
      sub(/^[[:space:]]+/, "", line)
      if (match(line, /^[^:=[:space:]]+/)) {
        print substr(line, RSTART, RLENGTH)
      }
    }
  ' "${file}" | sort -u
}

extract_missing_property_lines() {
  local missing_keys="$1"
  local source_file="$2"
  local output_file="$3"
  awk '
    BEGIN {
      while ((getline k < ARGV[1]) > 0) {
        wanted[k]=1
      }
      ARGV[1]=""
    }
    /^[[:space:]]*($|#|!)/ { next }
    {
      line=$0
      sub(/^[[:space:]]+/, "", line)
      if (match(line, /^[^:=[:space:]]+/)) {
        key=substr(line, RSTART, RLENGTH)
        if (key in wanted) {
          print $0
        }
      }
    }
  ' "${missing_keys}" "${source_file}" > "${output_file}"
}

check_config_keys() {
  if [[ "${SKIP_CONFIG_CHECK}" -eq 1 ]]; then
    echo "Skipping config key check."
    return
  fi

  local local_conf="${REPO_ROOT}/conf/application.properties"
  local remote_conf_ps="${REMOTE_DIR_PS}\\conf\\application.properties"
  local remote_conf_scp="${REMOTE_DIR_SCP}/conf/application.properties"
  local remote_conf_tmp="${TMP_DIR}/remote_application.properties"
  local local_keys="${TMP_DIR}/local_config_keys.txt"
  local remote_keys="${TMP_DIR}/remote_config_keys.txt"
  local missing_keys="${TMP_DIR}/missing_config_keys.txt"
  local missing_lines="${TMP_DIR}/missing_application.properties"

  if [[ ! -f "${local_conf}" ]]; then
    echo "Local config not found, skipping: ${local_conf}"
    return
  fi

  if ! remote_path_exists "${remote_conf_ps}"; then
    echo "Remote config not found, skipping key sync: ${remote_conf_ps}"
    return
  fi

  scp_from_remote "${remote_conf_scp}" "${remote_conf_tmp}" >/dev/null
  properties_keys "${local_conf}" > "${local_keys}"
  properties_keys "${remote_conf_tmp}" > "${remote_keys}"
  comm -23 "${local_keys}" "${remote_keys}" > "${missing_keys}"

  if [[ ! -s "${missing_keys}" ]]; then
    echo "Config key check passed: no new local keys."
    return
  fi

  echo "Config has local keys missing remotely:"
  sed 's/^/  - /' "${missing_keys}"

  if [[ "${DRY_RUN}" -eq 1 || "${CHECK_ONLY}" -eq 1 ]]; then
    echo "Dry/check mode: not appending config keys."
    return
  fi

  if ! confirm "Append these new key lines to remote conf/application.properties?"; then
    echo "Config key append skipped."
    return
  fi

  ensure_remote_dirs
  extract_missing_property_lines "${missing_keys}" "${local_conf}" "${missing_lines}"
  scp_to_remote "${missing_lines}" "${REMOTE_SYNC_DIR_SCP}/missing_application.properties" >/dev/null

  local target
  target="$(remote_literal "${REMOTE_DIR_PS}")"
  run_remote_ps "\$target='${target}'; \$conf=Join-Path \$target 'conf\application.properties'; \$missing=Join-Path \$target '_sync\missing_application.properties'; Add-Content -LiteralPath \$conf -Value ''; Add-Content -LiteralPath \$conf -Value ('# Added by sport-window-sync ' + (Get-Date -Format s)); Get-Content -LiteralPath \$missing | Add-Content -LiteralPath \$conf"
  echo "Config key append completed."
}

schema_dump() {
  local db_file="$1"
  sqlite3 -readonly "${db_file}" \
    "SELECT type || '|' || name || '|' || tbl_name || '|' || replace(replace(sql, char(13), ' '), char(10), ' ') FROM sqlite_schema WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY type, name;" \
    | tr -d '\r' \
    | sed 's/[[:space:]]\+/ /g'
}

remote_has_sqlite3() {
  run_remote_ps "$(remote_sqlite_command_ps) if (\$sqlite) { exit 0 } else { exit 1 }" >/dev/null 2>&1
}

remote_schema_dump() {
  local remote_db_ps="$1"
  local escaped_db escaped_query
  escaped_db="$(remote_literal "${remote_db_ps}")"
  escaped_query="$(remote_literal "SELECT type || '|' || name || '|' || tbl_name || '|' || replace(replace(sql, char(13), ' '), char(10), ' ') FROM sqlite_schema WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY type, name;")"
  run_remote_ps "$(remote_sqlite_command_ps) \$db='${escaped_db}'; \$query='${escaped_query}'; & \$sqlite -readonly \$db \$query" \
    | tr -d '\r' \
    | sed 's/[[:space:]]\+/ /g'
}

check_database_schema() {
  if [[ "${SKIP_DB_CHECK}" -eq 1 ]]; then
    echo "Skipping DB schema check."
    return
  fi

  local local_db_dir="${REPO_ROOT}/database"
  local ddl_diff_found=0
  local db local_schema remote_schema diff_file db_name remote_db_ps

  if [[ ! -d "${local_db_dir}" ]]; then
    echo "Local database directory not found, skipping DDL check: ${local_db_dir}"
    return
  fi

  if ! remote_has_sqlite3; then
    echo "Skipping DB schema check: remote sqlite3 is not available, and database file transfer is forbidden."
    return
  fi

  shopt -s nullglob
  local db_files=("${local_db_dir}"/*.db)
  shopt -u nullglob
  if [[ "${#db_files[@]}" -eq 0 ]]; then
    echo "No local database/*.db files found."
    return
  fi

  for db in "${db_files[@]}"; do
    db_name="$(basename "${db}")"
    remote_db_ps="${REMOTE_DIR_PS}\\database\\${db_name}"
    local_schema="${TMP_DIR}/local_${db_name}.schema"
    remote_schema="${TMP_DIR}/remote_${db_name}.schema"
    diff_file="${TMP_DIR}/${db_name}.ddl.diff"

    if ! remote_path_exists "${remote_db_ps}"; then
      echo "Remote DB missing or unreadable, no file replacement will be attempted: database/${db_name}"
      ddl_diff_found=1
      continue
    fi

    schema_dump "${db}" > "${local_schema}"
    remote_schema_dump "${remote_db_ps}" > "${remote_schema}"

    if ! diff -u "${remote_schema}" "${local_schema}" > "${diff_file}"; then
      ddl_diff_found=1
      echo "DDL differs for database/${db_name}:"
      sed -n '1,160p' "${diff_file}"
    else
      echo "DDL check passed: database/${db_name}"
    fi
  done

  if [[ "${ddl_diff_found}" -eq 0 ]]; then
    return
  fi

  echo "Database files were not replaced. Resolve DDL with migration SQL after explicit approval."
  echo "Database backup/upload/overwrite is forbidden in this script."
  if [[ "${CHECK_ONLY}" -eq 1 || "${DRY_RUN}" -eq 1 ]]; then
    return
  fi

  if [[ "${ASSUME_CONTINUE_WITHOUT_DB}" -eq 1 ]]; then
    echo "Continuing deploy without DB changes because --continue-without-db-sync was set."
    return
  fi

  if confirm "DDL differs. Continue syncing frontend/collect without changing database?"; then
    return
  fi

  echo "Stopped before file sync so DDL can be handled first."
  exit 1
}

make_payload() {
  local payload_dir="${TMP_DIR}/payload"
  mkdir -p "${payload_dir}"

  if [[ "${INCLUDE_EXE}" -eq 1 ]]; then
    cp "${REPO_ROOT}/${BUILD_EXE}" "${payload_dir}/main.exe"
  fi
  cp -a "${REPO_ROOT}/frontend" "${payload_dir}/frontend"
  cp -a "${REPO_ROOT}/collect" "${payload_dir}/collect"

  tar -C "${payload_dir}" -czf "${TMP_DIR}/${PAYLOAD_NAME}" .
  ensure_payload_has_no_database_files "${TMP_DIR}/${PAYLOAD_NAME}"
  echo "${TMP_DIR}/${PAYLOAD_NAME}"
}

ensure_payload_has_no_database_files() {
  local archive="$1"
  local bad_entries
  bad_entries="$(tar -tzf "${archive}" | awk '/(^|\/)[^\/]+\.db$/ || /^\.\/database\// { print }')"
  if [[ -n "${bad_entries}" ]]; then
    echo "Refusing to sync database files in payload:" >&2
    printf '%s\n' "${bad_entries}" >&2
    exit 1
  fi
}

remote_file_hash() {
  local ps_path="$1"
  local escaped
  escaped="$(remote_literal "${ps_path}")"
  run_remote_ps "\$ProgressPreference='SilentlyContinue'; \$p='${escaped}'; if (!(Test-Path -LiteralPath \$p)) { exit 3 }; (Get-FileHash -Algorithm SHA256 -LiteralPath \$p).Hash.ToLowerInvariant()" | tr -d '\r[:space:]'
}

remote_file_hash_maybe() {
  local ps_path="$1"
  local escaped
  escaped="$(remote_literal "${ps_path}")"
  run_remote_ps "\$ProgressPreference='SilentlyContinue'; \$p='${escaped}'; if (!(Test-Path -LiteralPath \$p)) { Write-Output '__MISSING__'; exit 0 }; (Get-FileHash -Algorithm SHA256 -LiteralPath \$p).Hash.ToLowerInvariant()" | tr -d '\r[:space:]'
}

local_file_hash() {
  sha256sum "$1" | awk '{print tolower($1)}'
}

sync_payload() {
  local archive="$1"
  local target
  local remote_items
  target="$(remote_literal "${REMOTE_DIR_PS}")"
  if [[ "${INCLUDE_EXE}" -eq 1 ]]; then
    remote_items="@('frontend','collect','main.exe')"
  else
    remote_items="@('frontend','collect')"
  fi

  ensure_remote_dirs
  echo "Uploading payload to ${USER_NAME}@${HOST}:${REMOTE_SYNC_DIR_SCP}/${PAYLOAD_NAME}"
  scp_to_remote "${archive}" "${REMOTE_SYNC_DIR_SCP}/${PAYLOAD_NAME}"

  echo "Applying payload on Windows host."
  run_remote_ps "\
\$ErrorActionPreference='Stop';
\$target='${target}';
\$sync=Join-Path \$target '_sync';
\$payload=Join-Path \$sync 'payload';
\$archive=Join-Path \$sync '${PAYLOAD_NAME}';
\$backup=Join-Path \$target ('_backup_' + (Get-Date -Format 'yyyyMMdd_HHmmss'));
\$items=${remote_items};
if (Test-Path -LiteralPath \$payload) { Remove-Item -Recurse -Force -LiteralPath \$payload };
New-Item -ItemType Directory -Force -Path \$payload | Out-Null;
tar -xzf \$archive -C \$payload;
New-Item -ItemType Directory -Force -Path \$backup | Out-Null;
foreach (\$name in \$items) {
  \$dst=Join-Path \$target \$name;
  if (Test-Path -LiteralPath \$dst) {
    Copy-Item -Recurse -Force -LiteralPath \$dst -Destination (Join-Path \$backup \$name);
  }
}
foreach (\$name in \$items) {
  \$dst=Join-Path \$target \$name;
  if (Test-Path -LiteralPath \$dst) { Remove-Item -Recurse -Force -LiteralPath \$dst };
  Copy-Item -Recurse -Force -LiteralPath (Join-Path \$payload \$name) -Destination \$dst;
}
Write-Output ('Backup: ' + \$backup)
"
}

verify_hashes() {
  local pairs=(
    "frontend/collect-ui/index.html|frontend\\collect-ui\\index.html"
    "collect/service_router.yml|collect\\service_router.yml"
    "collect/frontend/page_data/index.yml|collect\\frontend\\page_data\\index.yml"
    "collect/frontend/page_data/data/server/webshell_editor_pool_route.json|collect\\frontend\\page_data\\data\\server\\webshell_editor_pool_route.json"
    "collect/frontend/page_data/data/server/webshell_editor_pool_panel_fragment.json|collect\\frontend\\page_data\\data\\server\\webshell_editor_pool_panel_fragment.json"
    "collect/frontend/page_data/data/server/webshell_editor_pool_source_fragment.json|collect\\frontend\\page_data\\data\\server\\webshell_editor_pool_source_fragment.json"
    "collect/frontend/page_data/data/server/webshell_editor_pool_workspace_fragment.json|collect\\frontend\\page_data\\data\\server\\webshell_editor_pool_workspace_fragment.json"
    "collect/frontend/page_data/data/server/webshell_editor_pool_http_fragment.json|collect\\frontend\\page_data\\data\\server\\webshell_editor_pool_http_fragment.json"
    "collect/webshell/workspace_project/index.yml|collect\\webshell\\workspace_project\\index.yml"
    "collect/webshell/workspace_file/index.yml|collect\\webshell\\workspace_file\\index.yml"
    "collect/webshell/http_proxy/index.yml|collect\\webshell\\http_proxy\\index.yml"
    "collect/frontend/page_data/data/server/websql_config.json|collect\\frontend\\page_data\\data\\server\\websql_config.json"
    "collect/frontend/page_data/data/server/websql_pool.json|collect\\frontend\\page_data\\data\\server\\websql_pool.json"
    "collect/webshell/websql/index.yml|collect\\webshell\\websql\\index.yml"
  )
  local pair local_rel remote_rel local_path remote_path local_hash remote_hash
  local failures=0

  if [[ "${INCLUDE_EXE}" -eq 1 ]]; then
    pairs=("${BUILD_EXE}|main.exe" "${pairs[@]}")
  fi

  for pair in "${pairs[@]}"; do
    local_rel="${pair%%|*}"
    remote_rel="${pair#*|}"
    local_path="${REPO_ROOT}/${local_rel}"
    remote_path="${REMOTE_DIR_PS}\\${remote_rel}"

    if [[ ! -f "${local_path}" ]]; then
      echo "Local verify file missing: ${local_rel}" >&2
      failures=1
      continue
    fi

    local_hash="$(local_file_hash "${local_path}")"
    remote_hash="$(remote_file_hash_maybe "${remote_path}")"
    if [[ "${remote_hash}" == "__MISSING__" ]]; then
      echo "Remote verify file missing: ${remote_rel}" >&2
      failures=1
      continue
    fi
    if [[ "${local_hash}" != "${remote_hash}" ]]; then
      echo "Hash mismatch: ${local_rel} local=${local_hash} remote=${remote_hash}" >&2
      failures=1
      continue
    fi
    echo "Hash check passed: ${local_rel}"
  done

  if [[ "${failures}" -ne 0 ]]; then
    exit 1
  fi
}

restart_remote_app() {
  local target
  target="$(remote_literal "${REMOTE_DIR_PS}")"
  run_remote_ps "\
\$ErrorActionPreference='Stop';
\$target='${target}';
\$exe=Join-Path \$target 'main.exe';
\$escapedExe=[Regex]::Escape(\$exe);
\$processes=Get-CimInstance Win32_Process | Where-Object { \$_.ExecutablePath -and (\$_.ExecutablePath -ieq \$exe) };
foreach (\$p in \$processes) { Stop-Process -Id \$p.ProcessId -Force };
Start-Sleep -Seconds 1;
Start-Process -FilePath \$exe -WorkingDirectory \$target -WindowStyle Hidden;
Start-Sleep -Seconds 3;
\$listening=Get-NetTCPConnection -State Listen -LocalPort ${APP_PORT} -ErrorAction SilentlyContinue;
if (\$listening) { Write-Output 'Port ${APP_PORT} is listening.' } else { Write-Output 'Port ${APP_PORT} is not listening yet.' }
"
}

require_cmd sshpass
require_cmd ssh
require_cmd scp
require_cmd iconv
require_cmd base64
require_cmd awk
require_cmd sed
require_cmd sort
require_cmd comm
require_cmd diff
require_cmd tar
require_cmd sha256sum
if [[ "${INSTALL_REMOTE_SQLITE}" -eq 1 ]]; then
  require_cmd curl
  require_cmd openssl
fi
if [[ "${SKIP_DB_CHECK}" -eq 0 ]]; then
  require_cmd sqlite3
fi
if [[ "${INCLUDE_EXE}" -eq 1 && "${SKIP_BUILD}" -eq 0 && "${VERIFY_ONLY}" -eq 0 ]]; then
  require_cmd go
fi

TMP_DIR="$(mktemp -d "/tmp/sport-window-sync.XXXXXX")"

if [[ "${INSTALL_REMOTE_SQLITE}" -eq 1 ]]; then
  install_remote_sqlite3
  echo "Remote sqlite3 installation completed."
  exit 0
fi

if [[ "${VERIFY_ONLY}" -eq 1 ]]; then
  verify_hashes
  echo "Verify-only completed."
  exit 0
fi

if [[ "${CHECK_ONLY}" -eq 0 ]]; then
  build_windows_exe
fi

check_binary_rebuild_needed
check_database_schema
check_config_keys

if [[ "${CHECK_ONLY}" -eq 1 ]]; then
  echo "Check-only completed."
  exit 0
fi

if [[ "${DRY_RUN}" -eq 1 ]]; then
  echo "Dry run completed; no remote files were changed."
  exit 0
fi

payload_archive="$(make_payload)"
sync_payload "${payload_archive}"
verify_hashes

if confirm "Restart Windows program now?"; then
  restart_remote_app
else
  echo "Restart skipped. Files were replaced; remote app may still be running the previous executable until restarted."
fi

echo "Windows sync completed."
