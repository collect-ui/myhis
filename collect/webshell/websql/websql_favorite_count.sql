SELECT count(1) AS count
FROM websql_favorite a
WHERE 1=1
  AND COALESCE(a.is_delete, '0') = '0'
{{ if .project_code }}
  AND a.project_code = {{.project_code}}
{{ end }}
{{ if .item_type }}
  AND a.item_type = {{.item_type}}
{{ end }}
{{ if .path_exact }}
  AND a.path = {{.path_exact}}
{{ end }}
{{ if .folder_exact }}
  AND a.folder = {{.folder_exact}}
{{ end }}
{{ if .sql_text_exact }}
  AND a.sql_text = {{.sql_text_exact}}
{{ end }}
{{ if .exclude_id }}
  AND a.websql_favorite_id <> {{.exclude_id}}
{{ end }}
{{ if .keyword }}
  AND (
    a.name LIKE {{.keyword}}
    OR a.path LIKE {{.keyword}}
    OR a.folder LIKE {{.keyword}}
    OR a.sql_text LIKE {{.keyword}}
  )
{{ end }}
