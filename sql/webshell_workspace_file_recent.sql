CREATE TABLE IF NOT EXISTS webshell_workspace_file_recent (
  recent_id TEXT PRIMARY KEY,
  project_code TEXT NOT NULL,
  file_path TEXT NOT NULL,
  file_name TEXT,
  open_count INTEGER DEFAULT 1,
  last_open_time TEXT,
  create_time TEXT,
  create_user TEXT,
  modify_time TEXT,
  modify_user TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_workspace_file_recent_project_path
ON webshell_workspace_file_recent(project_code, file_path);

CREATE INDEX IF NOT EXISTS idx_workspace_file_recent_project_last_open
ON webshell_workspace_file_recent(project_code, last_open_time DESC);
