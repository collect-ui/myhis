select a.*
from work_task_module a
where 1=1
{{ if .search }}
and (
    a.name like {{.search}}
    or a.code like {{.search}}
)
{{ end}}
order by a.order_index