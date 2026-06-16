SELECT a.*
FROM webshell_workspace_project a
WHERE 1=1
  AND a.is_delete = '0'
{{ if .project_name }}
  AND a.project_name LIKE {{.project_name}}
{{ end }}
{{ if .project_code }}
  AND a.project_code = {{.project_code}}
{{ end }}
{{ if .project_code_keyword }}
  AND a.project_code LIKE {{.project_code_keyword}}
{{ end }}
{{ if .server_id }}
  AND a.server_id = {{.server_id}}
{{ end }}
{{ if .git_keyword }}
  AND a.git_repo_url LIKE {{.git_keyword}}
{{ end }}
{{ if .exclude_id }}
  AND a.webshell_workspace_project_id != {{.exclude_id}}
{{ end }}
{{ if .webshell_workspace_project_id }}
  AND a.webshell_workspace_project_id = {{.webshell_workspace_project_id}}
{{ end }}
{{ if .pagination }}
LIMIT {{.start}}, {{.size}}
{{ end }}
