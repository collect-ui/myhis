select  a.doc_project_content_id, a."group", a.doc_project_id,
a.create_time, a.create_user,
{{ if .with_content}}
a.doc_content,
{{ end }}
{{ if .with_children_content }}
(
  select group_concat(b.doc_content,char(10))
  from doc_project_content b
  where b.doc_project_content_id= a.doc_project_content_id or b.parent_id = a.doc_project_content_id
) as doc_content,
{{ end}}
a.topic, a.parent_id, a.order_index
from doc_project_content a
where 1=1
{{ if .doc_project_content_id}}
and a.doc_project_content_id = {{.doc_project_content_id}}
{{ end }}
{{ if .doc_project_id}}
and a.doc_project_id = {{.doc_project_id}}
{{ end }}
{{ if .tb_project_id}}
and exists(
    select 1
    from  doc_project t
    where t.doc_project_id = a.doc_project_id  and t.tb_project_id = {{.tb_project_id}}

)
{{ end }}
{{ if .group}}
and a.`group` = {{.group}}
{{ end }}
order by a.order_index

