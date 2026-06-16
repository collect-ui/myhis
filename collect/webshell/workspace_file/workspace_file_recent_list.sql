SELECT
  recent_id,
  project_code,
  file_path,
  file_name,
  open_count,
  last_open_time
FROM webshell_workspace_file_recent
WHERE project_code = {{.project_code}}
ORDER BY open_count DESC, last_open_time DESC
LIMIT 20
