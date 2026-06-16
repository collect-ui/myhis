select a.*
from sys_code a
where 1=1
{{ if .search }}
and (
        a.sys_code like {{.search}}
        or a.sys_code_text like {{.search}}
        or a.sys_code_type like {{.search}}
        or a.sys_code_type_name like {{.search}}
)
{{ end}}
order by a.sys_code_type, a.order_index