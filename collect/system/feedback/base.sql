select feedback_type.sys_code_text as feedback_type_name,a.*
from feedback a
left join sys_code feedback_type on feedback_type.sys_code_type = 'feedback_type' and feedback_type.sys_code = a.type
where 1=1
{{ if .feedback_id }}
and a.feedback_id = {{.feedback_id}}
{{ end}}
{{ if .search }}
and (a.contact like {{.search}} or a.title like {{.search}} or a.content like {{.search}})
{{ end}}
{{ if .type }}
and a.type = {{.type}}
{{ end }}

order by a.create_time desc
{{ if .pagination }}
limit {{.start}} , {{.size}}
{{ end }}