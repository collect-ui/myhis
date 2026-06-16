select a.*
from price_doc a
where 1=1
{{ if .search }}
and (
a.name like {{.search}}
or a.code like {{.search}}
)
{{ end }}
{{ if .project_id }}
and a.project_id = {{.project_id}}
{{ end}}
{{ if .doc_tree_id_list}}
and a.doc_tree_id in ({{.doc_tree_id_list}})
{{ else if .doc_tree_id }}
and a.doc_tree_id = {{.doc_tree_id}}
{{ end }}

{{ if .price_doc_tree_id }}
and a.price_doc_tree_id = {{.price_doc_tree_id}}
{{ end }}
{{ if .code }}
and a.code = {{.code}}
{{ end }}

{{if .code_list}}
and a.code in ({{.code_list}})
{{ end}}

{{ if .date_from }}
and a.date >= {{.date_from}}
{{ end}}



order by a.ancestors,a.order_index,a.code
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}