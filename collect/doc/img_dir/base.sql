select a.*
from img_dir a
where 1=1
{{ if .img_dir_id }}
and a.img_dir_id = {{.img_dir_id}}
{{ end }}
{{ if .search }}
and ( a.name like {{.search}} )
{{ end }}
{{ if .code}}
and a.code = {{.code}}
{{ end }}
{{ if .exclude}}
and a.img_dir_id != {{.exclude}}
{{ end }}
order by a.order_index