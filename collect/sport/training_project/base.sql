select a.*
from training_project a
where a.is_delete = '0'
{{ if .training_project_id }}
and a.training_project_id = {{.training_project_id}}
{{ end}}
{{ if .search }}
and a.name like {{.search}}
{{ end}}
{{ if .body_part }}
and a.body_part = {{.body_part }}
{{ end }}
{{ if  .training_type  }}
and a.training_type = {{.training_type }}
{{ end }}
{{ if  .difficulty_level  }}
and a.difficulty_level = {{.difficulty_level }}
{{ end }}
{{ if .training_project_code }}
and a.training_project_code = {{.training_project_code }}
{{ end }}
{{ if .exclude }}
and a.training_project_id != {{.exclude}}
{{ end}}
order by a.order_index
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}