select a.*
from price_doc_qing_project a
where 1=1
{{if .price_doc_qing_project_id }}
and a.price_doc_qing_project_id = {{.price_doc_qing_project_id}}
{{ end }}
order by a.order_index