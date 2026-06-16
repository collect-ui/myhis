CREATE TABLE IF NOT EXISTS webshell_workspace_project (
  webshell_workspace_project_id TEXT PRIMARY KEY,
  project_name TEXT NOT NULL,
  project_code TEXT NOT NULL,
  order_id INTEGER NOT NULL DEFAULT 1,
  show_home TEXT NOT NULL DEFAULT '1',
  server_id TEXT NOT NULL,
  server_os_users_id TEXT NOT NULL,
  project_dir TEXT NOT NULL,
  git_repo_url TEXT NOT NULL,
  is_delete TEXT NOT NULL DEFAULT '0',
  create_time TEXT NOT NULL,
  create_user TEXT,
  modify_time TEXT NOT NULL,
  modify_user TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_webshell_workspace_project_code
ON webshell_workspace_project(project_code);

CREATE INDEX IF NOT EXISTS idx_webshell_workspace_project_server_id
ON webshell_workspace_project(server_id);

CREATE INDEX IF NOT EXISTS idx_webshell_workspace_project_order_id
ON webshell_workspace_project(order_id);
