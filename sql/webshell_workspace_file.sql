CREATE TABLE IF NOT EXISTS webshell_workspace_file (
  file_id TEXT PRIMARY KEY,
  project_code TEXT NOT NULL,
  name TEXT NOT NULL,
  path TEXT NOT NULL,
  parent_id TEXT,
  is_dir TEXT NOT NULL DEFAULT '1',
  is_delete TEXT NOT NULL DEFAULT '0',
  create_time TEXT NOT NULL,
  create_user TEXT,
  modify_time TEXT NOT NULL,
  modify_user TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_webshell_workspace_file_project_path
ON webshell_workspace_file(project_code, path);

CREATE INDEX IF NOT EXISTS idx_webshell_workspace_file_project_code
ON webshell_workspace_file(project_code);

CREATE INDEX IF NOT EXISTS idx_webshell_workspace_file_parent_id
ON webshell_workspace_file(parent_id);
