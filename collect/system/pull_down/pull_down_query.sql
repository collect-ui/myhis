select a.*
from pull_down a
where 1=1
{{ if .code }}
and a.code = {{.code}}
{{ end }}
{{ if .pull_down_id }}
and a.pull_down_id = {{.pull_down_id}}
{{ end}}
{{ if .exclude}}
 and a.pull_down_id != {{.exclude}}
{{ end}}
order by a.`group`,a.order_index