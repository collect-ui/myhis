---
name: sport-window-sync
description: Use when deploying this sport/moon Go low-code app to the fixed Windows SSH environment at 10.96.32.180, syncing frontend/collect safely, checking config drift, and asking before exe replacement, config sync, or restart. Database file backup/upload/sync is forbidden.
---

# Sport Windows Sync

Use this skill when the user asks to sync or deploy this repository to the Windows program environment.

## Fixed Environment

- Local repo: `/data/project/sport`
- Remote host: `10.96.32.180`
- Remote user: `18874948657`
- Remote dir: `F:\moon-windows-amd64-clean`
- Main artifacts: `frontend/`, `collect/`
- Optional executable artifact: `windows/main.exe` only after explicit confirmation
- App port convention: `8009`
- Remote sqlite3 location for read-only DDL checks: `F:\moon-windows-amd64-clean\tools\sqlite\sqlite3.exe`

Set the SSH password in the shell instead of writing it into files:

```bash
export SPORT_WINDOW_SSH_PASSWORD='...'
```

Fallback variable: `WINDOW_SERVER_PASSWORD`.

## Commands

Run checks without modifying the Windows host:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --dry-run
```

Default sync: check config drift and sync only `frontend/` and `collect/`, then ask before restart. This workflow must never build or replace `main.exe` unless explicitly requested, and must never back up, upload, overwrite, or copy database files. If Go/model/plugin changes are present, especially new database tables or fields, do not use default file-only sync unless the operator explicitly accepts that the remote executable will stay unchanged:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh
```

Executable sync is opt-in. Ask the user before using this; when approved, include `windows/main.exe` in build/sync/verify:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --include-exe
```

Install or upgrade remote SQLite CLI for read-only DDL comparison. This downloads the official Windows x64 SQLite tools locally, verifies SHA3, uploads only the tools zip to remote `_sync/`, installs `sqlite3.exe` under `tools\sqlite`, then exits. It must not transfer any `database/*.db` files:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --install-remote-sqlite
```

After installation, run:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --check-only
```

The sync script first tries `SPORT_WINDOW_REMOTE_SQLITE3`, then `F:\moon-windows-amd64-clean\tools\sqlite\sqlite3.exe`, then Windows `PATH`. Use `SPORT_WINDOW_SQLITE_TOOLS_URL` and `SPORT_WINDOW_SQLITE_TOOLS_SHA3` only when the official SQLite download version changes.

Use executable sync when the local changes add or modify:

- Go models or table registration under `model/`
- plugin handlers or registration under `plugins/`
- startup/runtime wiring in `main.go`
- database-backed behavior that depends on new tables, columns, or handlers

For these changes, `collect/` and `frontend/` alone can make the Windows UI point at endpoints or fields that the old `main.exe` does not know about.

Check only database DDL and config-key drift. DDL checks are best-effort and must not transfer full `.db` files; if remote read-only SQL access is unavailable, report that DB checking was skipped:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --check-only
```

Check whether the deployed Windows files match local frontend/config without modifying the remote host:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --verify-only
```

Include executable hash in verification only when explicitly needed:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --verify-only --include-exe
```

Skip slow or already-handled phases when appropriate:

```bash
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --skip-build
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --skip-db-check
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --skip-config-check
bash .codex/skills/sport-window-sync/scripts/sync_window_env.sh --allow-file-only-with-go-changes
```

## Safety Rules

- Do not back up, upload, overwrite, copy, scp, or otherwise transfer `database/*.db`. Database file operations are forbidden in this skill, including "temporary" migration backups and `_sync` uploads.
- Do not build, upload, overwrite, or restart `main.exe` by default. Ask the user whether executable replacement is needed; most syncs should replace only `frontend/` and `collect/`.
- When Go/model/plugin files changed, treat file-only sync as risky. New database fields/tables usually require a freshly built Windows executable because model registration, migrations, and plugin handlers are compiled into `main.exe`.
- Before deployment, compare local and remote SQLite schemas only when this can be done without transferring database files. If remote read-only SQL access is unavailable, report that DDL checking was skipped and keep the deployment file-only.
- Installing remote sqlite3 is allowed because it installs tools only. It does not authorize backing up, downloading, uploading, copying, or overwriting any database file.
- If DDL differs, show the diff and stop before any database work. This skill may provide migration SQL text, but it must not apply SQL, back up DB files, or move DB files.
- Do not overwrite `conf/application.properties`. Compare keys only; if local has new keys, ask before appending those key lines to the remote config.
- Do not restart the Windows program automatically. After file replacement, ask the user whether to restart.
- Always stage files under remote `_sync/` first and keep a timestamped remote backup of replaced `frontend/` and `collect/`; include `main.exe` in the backup only when executable replacement was explicitly approved.
- Treat `database/`, `conf/`, remote credentials, and local runtime outputs as environment state, not source artifacts.

## Workflow

1. Confirm the current working tree and whether uncommitted changes are intended for deployment.
2. Check whether binary-relevant files changed: `main.go`, `go.mod`, `go.sum`, `model/**/*.go`, and `plugins/**/*.go`. If they changed, explain that new DB tables/fields and handlers require executable sync, then ask whether `main.exe` replacement is approved. Default to no for unrelated UI/config-only changes. Build the Windows executable with `GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o windows/main.exe main.go` only when the user approves exe replacement or passes `--include-exe`.
3. Compare SQLite DDL for `database/*.db` only through read-only SQL access on each side. Do not scp, back up, upload, or overwrite `.db` files. If read-only remote SQL access is unavailable, skip DDL comparison with a clear warning and keep the deployment file-only.
4. Compare `conf/application.properties` keys. Append new local keys to the remote config only after explicit confirmation.
5. Package and upload only `frontend/` and `collect/` by default; include `main.exe` only with explicit approval.
6. Replace remote files from staging, verify probe-file hashes, including `webshell-editor-pool` and `websql-pool` page/config probes, then ask whether to restart.
7. If restart is approved, stop the existing `main.exe`, start it from the remote dir, and verify the port/process state.

## Recommended Use

- First-time or broken DDL checking: run `--install-remote-sqlite`, then `--check-only`.
- UI/config-only changes under `frontend/` or `collect/`: run `--dry-run`, then the default sync command.
- Go/model/plugin/database-backed changes: run `--check-only`; resolve any real DDL migration outside this skill; then run `--include-exe` so `main.exe`, `frontend/`, and `collect/` are deployed together.
- If `--check-only` reports only harmless SQLite DDL text differences, such as equivalent column definitions in a different order, note the residual diff and continue only after the operator accepts it.
- After sync, use `--verify-only`; add `--include-exe` when the executable was part of the deployment.
- For WebShell/WebSQL work, validate the local pages before deploying when possible:
  - `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`
  - `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool`

## Notes

- The helper script intentionally does not auto-apply database migrations. If DDL sync is needed, produce migration SQL for a separate operator-owned DB process outside this skill. Do not back up, copy, or upload database files from this workflow.
- The helper script uses Windows PowerShell over SSH and `scp`; it does not require rsync on the Windows host.
- For local HTTP checks against this project, use `curl --noproxy '*'` because this environment may define HTTP proxy variables.
