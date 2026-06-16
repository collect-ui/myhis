select a.*
from role a
where 1=1
{{ if .search }}
and (
a.role_name like {{.search}}
or a.role_code like {{.search}}
)
{{ end }}
{{ if .role_id_list }}
and a.role_id in ({{.role_id_list}})
{{ end}}
{{ if .sys_menu_id }}
and exists(
    select 1
    from role_menu b
    where b.role_id = a.role_id
    and b.sys_menu_id = {{.sys_menu_id }}
)
{{ end}}

order by a.order_index
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}