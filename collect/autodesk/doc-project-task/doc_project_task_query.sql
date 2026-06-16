select a.*
from doc_project_task a
left join doc_project b on a.doc_project_id = b.doc_project_id
where 1=1
{{ if .tb_project_id }}
and b.tb_project_id = {{.tb_project_id}}
{{ end}}
{{ if .doc_project_id}}
  and a.doc_project_id = {{.doc_project_id }}
{{ end }}
{{ if .code }}
and a.code = {{.code}}
{{ end}}

{{ if .status}}
and a.status = {{.status}}
{{ end }}
order by a.create_time
{{ if .size }}
limit {{.size }}
{{ end }}