SELECT
  a.quick_text_id,
  a.title,
  a.description,
  a.content,
  a.role_type,
  a.effect_exts,
  COALESCE(a.use_count, 0) AS use_count,
  COALESCE(a.is_favorite, '0') AS is_favorite,
  a.create_time,
  a.create_user,
  a.modify_time,
  a.modify_user
FROM webshell_quick_text a
WHERE 1=1
  AND COALESCE(a.is_delete, '0') = '0'
{{ if .keyword }}
  AND (
    a.title LIKE {{.keyword}}
    OR a.description LIKE {{.keyword}}
    OR a.content LIKE {{.keyword}}
  )
{{ end }}
{{ if .role_type }}
  AND a.role_type = {{.role_type}}
{{ end }}
{{ if .favorite_only }}
  AND a.is_favorite = '1'
{{ end }}
{{ if .effect_ext }}
  AND (
    COALESCE(a.effect_exts, '') = ''
    OR lower(replace(a.effect_exts, ' ', '')) = '*'
    OR (',' || lower(replace(a.effect_exts, ' ', '')) || ',') LIKE ('%,' || lower({{.effect_ext}}) || ',%')
    OR (
      lower({{.effect_ext}}) = 'readme'
      AND lower(replace(a.effect_exts, ' ', '')) LIKE '%readme%'
    )
  )
{{ end }}
ORDER BY
  COALESCE(a.use_count, 0) DESC,
  COALESCE(a.is_favorite, '0') DESC,
  a.modify_time DESC,
  a.create_time DESC
{{ if .pagination }}
LIMIT {{.start}}, {{.size}}
{{ end }}
