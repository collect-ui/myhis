SELECT count(1) AS count
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
