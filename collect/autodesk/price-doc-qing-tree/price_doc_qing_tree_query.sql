select a.*
from  price_doc_qing_tree a
where 1=1
{{ if .search }}
and a.name like {{ .search }}
{{ end }}

{{  if .project_id }}
and a.project_id = {{.project_id}}
{{ end }}
{{ if .project_name }}
and exists(
    select 1
    from  price_doc_qing_project b
    where b.price_doc_qing_project_id = a.project_id
    and b.project_name = {{.project_name }}
)
{{ end }}
order by a.order_index,a.name