select a.*
from schema_page a
where a.belong_project ={{.belong_project}}
{{ if .code }}
and a.code = {{.code}}
{{ end}}

{{if .schema_page_id}}
and a.schema_page_id = {{.schema_page_id}}
{{end}}

{{ if .exclude}}
and a.schema_page_id != {{.exclude}}
{{ end}}

order by order_index+0