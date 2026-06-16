select a.*
from doc_project a
where 1=1
{{ if .doc_project_id}}
and a.doc_project_id = {{.doc_project_id}}
{{ end }}
{{ if .tb_project_id}}
and a.tb_project_id = {{.tb_project_id}}
{{ end }}
{{ if .tb_project_id_list}}
and a.tb_project_id in ({{.tb_project_id_list }})
{{ end }}

{{ if .name }}
 and a.name = {{.name}}
{{ end }}
{{ if .code }}
 and a.name = {{.code}}
{{ end }}
{{ if .exclude}}
and a.doc_project_id != {{ .exclude }}
{{ end }}
{{ if .search }}
and  a.name like {{.search}}
{{ end}}
