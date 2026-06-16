select a.*
from tb_config a
where 1=1
{{ if .key }}
 and a.key = {{.key}}
{{ end }}
{{ if .exclude}}
and a.tb_config_id != {{ .exclude }}
{{ end }}
{{ if .search }}
and  (a.key like {{.search}} or a.value like {{.search}} or a.description like {{.search}})
{{ end}}
{{ if .key_list}}
and a.key in ({{.key_list}})
{{ end}}
{{ if .tb_project_id }}
and a.tb_project_id = {{.tb_project_id}}
{{ end}}
{{ if .project_name }}
and exists(
    select 1
    from tb_project t
    where t.tb_project_id = a.tb_project_id and t.name = {{.project_name}}
)
{{ end }}

{{ if .fields}}
and a.key in ({{.fields}})
{{ end}}

{{ if .schema_page_code}}
 and exists (
     select 1
     from schema_page_field f
     where  a.key = f.key and f.schema_page_code = {{.schema_page_code}}
     {{ if  .only_attachments }}
     and f.tag = 'attachment' and a.value!=''
     {{ end }}
)
{{ end}}

order by a.order_index