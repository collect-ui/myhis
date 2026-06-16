select a.*
from attachment a
where 1=1
{{ if .search }}
and ( a.title LIKE {{.search}} or a.filename LIKE {{.search}})
{{ end }}
{{ if .filetype }}
and a.filetype = {{.filetype }}
{{ end }}


order by a.order_index