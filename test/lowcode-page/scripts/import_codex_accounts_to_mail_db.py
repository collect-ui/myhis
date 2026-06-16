#!/usr/bin/env python3
"""Update mail_account codex fields from ~/.codex/accounts (update-only by default)."""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import json
import sqlite3
from pathlib import Path
from typing import Any


def build_parser() -> argparse.ArgumentParser:
    repo_root = Path(__file__).resolve().parents[3]
    parser = argparse.ArgumentParser(description="Update mail_account from codex accounts")
    parser.add_argument(
        "--accounts-dir",
        default=str(Path.home() / ".codex" / "accounts"),
        help="Path to codex accounts directory (default: ~/.codex/accounts)",
    )
    parser.add_argument("--registry", default="registry.json", help="Registry filename")
    parser.add_argument(
        "--db",
        default=str(repo_root / "database" / "price.db"),
        help="SQLite db path (default: <repo>/database/price.db)",
    )
    parser.add_argument("--limit", type=int, default=0, help="Only process first N accounts")
    parser.add_argument("--dry-run", action="store_true", help="Preview only")
    parser.add_argument(
        "--allow-insert",
        action="store_true",
        help="Insert when no match found (default: disabled)",
    )
    return parser


def read_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def account_key_to_auth_file(account_key: str) -> str:
    encoded = base64.b64encode(account_key.encode("utf-8")).decode("ascii")
    return f"{encoded.rstrip('=')}.auth.json"


def local_now() -> str:
    return dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def proton_to_outlook(email: str) -> str:
    if "@" not in email:
        return ""
    local = email.split("@", 1)[0]
    return f"{local}@outlook.com"


def get_existing_columns(conn: sqlite3.Connection, table: str) -> set[str]:
    rows = conn.execute(f"PRAGMA table_info({table});").fetchall()
    return {row[1] for row in rows}


def choose_match(conn: sqlite3.Connection, proton_email: str) -> sqlite3.Row | None:
    local = proton_email.split("@", 1)[0] if "@" in proton_email else proton_email
    outlook_email = proton_to_outlook(proton_email)

    sql = """
    SELECT *
    FROM mail_account
    WHERE ifnull(is_delete,'0')='0'
      AND (
        proton_email = ?
        OR email_name = ?
        OR email_name = ?
        OR email_name = ?
        OR password = ?
        OR password = ?
      )
    ORDER BY
      CASE
        WHEN proton_email = ? AND email_name LIKE '%@outlook.com' THEN 0
        WHEN proton_email = ? THEN 1
        WHEN email_name = ? THEN 2
        WHEN email_name = ? THEN 3
        ELSE 9
      END,
      order_index ASC,
      create_time DESC
    LIMIT 1
    """
    return conn.execute(
        sql,
        (
            proton_email,
            proton_email,
            outlook_email,
            local,
            proton_email,
            local,
            proton_email,
            proton_email,
            outlook_email,
            proton_email,
        ),
    ).fetchone()


def find_password_matches(conn: sqlite3.Connection, proton_email: str) -> list[sqlite3.Row]:
    local = proton_email.split("@", 1)[0] if "@" in proton_email else proton_email
    sql = """
    SELECT *
    FROM mail_account
    WHERE ifnull(is_delete,'0')='0'
      AND email_name LIKE '%@outlook.com'
      AND password = ?
    ORDER BY order_index ASC, create_time DESC
    """
    return conn.execute(sql, (local,)).fetchall()


def build_update_fields(account: dict[str, Any], auth: dict[str, Any], auth_file: str) -> dict[str, Any]:
    tokens = auth.get("tokens") or {}
    account_id = tokens.get("account_id") or account.get("chatgpt_account_id") or ""
    plan = str(account.get("plan") or "")

    return {
        "proton_email": account.get("email") or "",
        "codex_device_auth_id": auth.get("device_auth_id") or "",
        "codex_authorization_code": auth.get("authorization_code") or "",
        "codex_code_verifier": auth.get("code_verifier") or "",
        "codex_access_token": tokens.get("access_token") or "",
        "codex_refresh_token": tokens.get("refresh_token") or "",
        "codex_id_token": tokens.get("id_token") or "",
        "codex_token_type": tokens.get("token_type") or "",
        "codex_scope": tokens.get("scope") or "",
        "codex_expires_in": str(tokens.get("expires_in") or ""),
        "codex_expires_at": str(tokens.get("expires_at") or ""),
        "codex_account_id": account_id,
        "codex_auth_json": json.dumps(auth, ensure_ascii=False),
        "codex_auth_status": "token_ready" if tokens.get("access_token") else "",
        "codex_auth_msg": f"updated from ~/.codex/accounts/{auth_file}",
        "codex_last_auth_time": (auth.get("last_refresh") or "").strip() or local_now(),
        "codex_usage_plan_type": plan,
    }


def insert_row(conn: sqlite3.Connection, proton_email: str, fields: dict[str, Any], columns: set[str]) -> None:
    # Optional path; default workflow should not use insert.
    data = {
        "mail_account_id": fields.get("codex_account_id") or proton_email,
        "order_index": 999999,
        "email_name": proton_email,
        "password": "imported_from_codex",
        "guid_code": "",
        "recovery_code": "",
        "raw_text": json.dumps({"source": "~/.codex/accounts"}, ensure_ascii=False),
        "is_current_running": "0",
        "current_run_mark_time": "",
        "proton_registered": "0",
        "proton_password": "Zhangzhi@888",
        "create_time": local_now(),
        "create_user": "codex-import",
        "is_delete": "0",
    }
    data.update(fields)
    data = {k: v for k, v in data.items() if k in columns}

    cols = list(data.keys())
    placeholders = ",".join(["?" for _ in cols])
    sql = f"INSERT INTO mail_account ({','.join(cols)}) VALUES ({placeholders})"
    conn.execute(sql, tuple(data[c] for c in cols))


def update_row(conn: sqlite3.Connection, mail_account_id: str, fields: dict[str, Any], columns: set[str]) -> None:
    update_fields = {k: v for k, v in fields.items() if k in columns}
    sets = ", ".join([f"{k}=?" for k in update_fields.keys()])
    sql = f"UPDATE mail_account SET {sets} WHERE mail_account_id=?"
    conn.execute(sql, tuple(update_fields.values()) + (mail_account_id,))


def main() -> int:
    args = build_parser().parse_args()

    accounts_dir = Path(args.accounts_dir).expanduser().resolve()
    registry_path = accounts_dir / args.registry
    db_path = Path(args.db).expanduser().resolve()

    if not registry_path.exists():
        raise SystemExit(f"registry not found: {registry_path}")

    registry = read_json(registry_path)
    accounts = list(registry.get("accounts") or [])
    if args.limit and args.limit > 0:
        accounts = accounts[: args.limit]

    conn = sqlite3.connect(str(db_path))
    conn.row_factory = sqlite3.Row

    updated = 0
    inserted = 0
    missing_auth = 0
    unmatched: list[str] = []

    try:
        cols = get_existing_columns(conn, "mail_account")

        for account in accounts:
            proton_email = (account.get("email") or "").strip().lower()
            if not proton_email:
                continue
            account_key = account.get("account_key") or ""
            auth_file = account_key_to_auth_file(account_key)
            auth_path = accounts_dir / auth_file
            if not auth_path.exists():
                missing_auth += 1
                continue
            auth = read_json(auth_path)
            fields = build_update_fields(account, auth, auth_file)

            primary = choose_match(conn, proton_email)
            password_rows = find_password_matches(conn, proton_email)
            targets: list[sqlite3.Row] = []
            seen: set[str] = set()
            if primary is not None:
                targets.append(primary)
                seen.add(primary["mail_account_id"])
            for row in password_rows:
                pk = row["mail_account_id"]
                if pk not in seen:
                    targets.append(row)
                    seen.add(pk)

            if not targets:
                if args.allow_insert:
                    if not args.dry_run:
                        insert_row(conn, proton_email, fields, cols)
                    inserted += 1
                else:
                    unmatched.append(proton_email)
                continue

            if not args.dry_run:
                for row in targets:
                    update_row(conn, row["mail_account_id"], fields, cols)
            updated += len(targets)

        if not args.dry_run:
            conn.commit()

        print(f"Updated: {updated}")
        print(f"Inserted: {inserted}")
        print(f"Missing auth files: {missing_auth}")
        print(f"Unmatched: {len(unmatched)}")
        if unmatched:
            for e in unmatched[:20]:
                print(f"  - {e}")
        if args.dry_run:
            print("Dry-run enabled, no DB write.")
        return 0
    finally:
        conn.close()


if __name__ == "__main__":
    raise SystemExit(main())
