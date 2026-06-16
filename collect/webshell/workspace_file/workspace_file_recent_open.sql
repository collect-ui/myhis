INSERT INTO webshell_workspace_file_recent (
  recent_id,
  project_code,
  file_path,
  file_name,
  open_count,
  last_open_time,
  create_time,
  create_user,
  modify_time,
  modify_user
) VALUES (
  {{.recent_id}},
  {{.project_code}},
  {{.file_path}},
  {{.file_name}},
  1,
  {{.now_time}},
  {{.now_time}},
  {{if .session_user_id}}{{.session_user_id}}{{else}}''{{end}},
  {{.now_time}},
  {{if .session_user_id}}{{.session_user_id}}{{else}}''{{end}}
)
ON CONFLICT(project_code, file_path) DO UPDATE SET
  file_name = excluded.file_name,
  open_count = COALESCE(webshell_workspace_file_recent.open_count, 0) + 1,
  last_open_time = excluded.last_open_time,
  modify_time = excluded.modify_time,
  modify_user = excluded.modify_user;
