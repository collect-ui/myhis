select a.*
from work_task_version a
where 1=1
{{ if .search }}
and a.name like {{.search}} or a.code like {{.search}}
{{ end }}
order by a.create_time desc
limit 10
