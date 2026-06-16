DELETE FROM webshell_workspace_file_recent
WHERE project_code = {{.project_code}}
  AND recent_id = {{.recent_id}}
