select -- b.doc_content,b.topic,
a.*
from doc_project_detail a
left join doc_project p on a.doc_project_id = p.doc_project_id
left join tb_project p2 on p.tb_project_id = p2.tb_project_id
--left join doc_project_content b on a.doc_project_id = b.doc_project_id and a.`group` = b.`group`
where 1=1
{{ if .tb_project_id }}
and p2.tb_project_id = {{.tb_project_id}}
{{ end }}
{{ if .name }}
 and a.name = {{.name}}
{{ end }}
{{ if .detail }}
 and a.detail = {{.detail}}
{{ end }}
{{ if .doc_project_detail_id }}
and a.doc_project_detail_id = {{.doc_project_detail_id}}
{{ end}}

{{ if .exclude}}
and a.doc_project_detail_id != {{ .exclude }}
{{ end }}
{{ if .search }}
and  (a.name like {{.search}} or a.code like {{.search}} or a.detail like {{.search}})
{{ end}}
{{ if .doc_project_id }}
and a.doc_project_id = {{.doc_project_id}}
{{ end}}

{{ if .no_doc_content}}
and ifnull(a.doc_content,'') =''
{{end}}
{{ if .has_doc_content}}
and ifnull(a.doc_content,'') !=''
{{end}}
{{ if .has_group}}
and ifnull(a.`group`,'') !='' and a.`group` !='0'
{{end}}
{{ if .no_group}}
and a.`group` is null
{{end}}
order by a.order_index
{{if  .size}}
limit {{.size}}
{{ end }}