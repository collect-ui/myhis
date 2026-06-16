select a.*
from registration a
where 1=1
{{ if .registration_id }}
and a.registration_id = {{.registration_id}}
{{ end}}
{{ if .search }}
and (a.username like {{.search}} or a.company like {{.search}} or a.phone like {{.search}})
{{ end}}
{{ if .phone }}
and a.phone = {{.phone}}
{{ end }}
{{ if .username }}
and a.username = {{.username}}
{{ end }}
{{ if .status }}
and a.status = {{.status}}
{{ end }}
{{ if .status_list }}
and a.status in ({{.status_list}})
{{ end }}
{{ if .exclude }}
and a.registration_id != {{.exclude}}
{{ end}}
order by
  CASE a.status
    WHEN 'pending' THEN 1
    WHEN 'success' THEN 2
    WHEN 'fail' THEN 3
    ELSE 4
  END,
a.create_time desc
{{ if .pagination }}
limit {{.start}} , {{.size}}
{{ end }}