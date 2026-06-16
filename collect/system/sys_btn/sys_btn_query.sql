select a.*
from sys_btn a
where 1=1
{{ if .menu_code }}
and a.menu_code = {{.menu_code}}
{{ end }}
order by a.order_index
