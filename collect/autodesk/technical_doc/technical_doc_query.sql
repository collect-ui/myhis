select a.*
from technical_doc a
where 1=1
{{ if .technical_doc_id }}
and a.technical_doc_id = {{.technical_doc_id }}
{{ end}}
{{ if .name }}
and a.name = {{.name }}
{{ end}}
{{ if .exclude}}
and a.technical_doc_id != {{.exclude }}
{{ end }}
{{ if .parent_id }}
and a.parent_id = {{.parent_id }}
{{ end }}




{{ if .search }}
and a.name like {{.search}}
{{ end }}
{{ if .type  }}
and a.type = {{.type }}
{{ end}}
order by a.type,a.order_index,a.name