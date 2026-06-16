WITH project_root AS (
  SELECT p.project_code, p.project_dir
  FROM webshell_workspace_project p
  WHERE p.is_delete = '0'
    AND p.webshell_workspace_project_id = (
      SELECT p2.webshell_workspace_project_id
      FROM webshell_workspace_project p2
      WHERE p2.project_code = p.project_code
        AND p2.is_delete = '0'
      ORDER BY p2.modify_time DESC
      LIMIT 1
    )
)
SELECT
  a.*,
  CASE
    WHEN ifnull(pr.project_dir, '') != ''
      AND a.path LIKE pr.project_dir || '/%'
      THEN substr(a.path, length(pr.project_dir) + 1)
    ELSE a.path
  END AS path_display
FROM webshell_workspace_file a
LEFT JOIN project_root pr ON pr.project_code = a.project_code
WHERE 1=1
  AND a.is_delete = '0'
{{ if .project_code }}
  AND a.project_code = {{.project_code}}
{{ end }}
{{ if .name }}
  AND a.name LIKE {{.name}}
{{ end }}
{{ if .keyword }}
  AND (
    a.name LIKE {{.keyword}}
    OR (
      CASE
        WHEN ifnull(pr.project_dir, '') != ''
          AND a.path LIKE pr.project_dir || '/%'
          THEN substr(a.path, length(pr.project_dir) + 1)
        ELSE a.path
      END
    ) LIKE {{.keyword}}
  )
{{ end }}
{{ if .parent_id }}
  AND a.parent_id = {{.parent_id}}
{{ end }}
{{ if .root_only }}
  AND (a.parent_id = '' OR a.parent_id = '0' OR a.parent_id IS NULL)
{{ end }}
{{ if .is_dir }}
  AND a.is_dir = {{.is_dir}}
{{ end }}
{{ if .path_exact }}
  AND a.path = {{.path_exact}}
{{ end }}
{{ if .exclude_file_id }}
  AND a.file_id != {{.exclude_file_id}}
{{ end }}
{{ if .pagination }}
LIMIT {{.start}}, {{.size}}
{{ end }}
