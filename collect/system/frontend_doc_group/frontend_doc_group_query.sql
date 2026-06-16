select a.*
from frontend_doc_group a
where belong_project = {{.belong_project}}
{{ if .search }}
and (a.name like {{.search}} or a.code like {{.search}})
{{ end}}
{{ if .frontend_doc_group_id }}
and a.frontend_doc_group_id = {{.frontend_doc_group_id}}
{{ end }}
{{ if .type }}
and a.type = {{.type}}
{{ end}}
{{ if .code }}
and a.code = {{.code}}
{{ end }}
{{ if .exclude }}
and a.frontend_doc_group_id != {{.exclude}}
{{ end }}
order by a.order_index