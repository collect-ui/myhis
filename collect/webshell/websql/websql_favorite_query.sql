SELECT
  a.websql_favorite_id,
  a.websql_favorite_id AS id,
  a.project_code,
  a.item_type,
  a.item_type AS type,
  a.name,
  a.name AS title,
  a.path,
  a.folder,
  a.sql_text,
  a.sql_text AS sql,
  a.connection_id,
  a.connection_name,
  a.connection_name AS connection,
  a.driver,
  a.is_delete,
  a.create_time,
  a.create_time AS created_at,
  a.modify_time,
  a.modify_time AS updated_at,
  a.create_user,
  a.modify_user
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
ORDER BY
  CASE WHEN a.item_type = 'folder' THEN 0 ELSE 1 END ASC,
  COALESCE(NULLIF(a.path, ''), a.folder, '') ASC,
  a.name ASC,
  a.modify_time DESC
{{ if .pagination }}
LIMIT {{.start}}, {{.size}}
{{ end }}
