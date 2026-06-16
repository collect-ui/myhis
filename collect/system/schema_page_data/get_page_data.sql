select a.*
from schema_page_data a
where 1=1
{{ if .schema_page_code}}
and a.schema_page_code ={{.schema_page_code}}
{{ end}}
{{ if .belong_id }}
and a.belong_id = {{.belong_id}}
{{ end }}
{{ if .belong_id_list }}
and a.belong_id in ({{.belong_id_list}})
{{ end }}
order by a.parent_key,a.`index`