select a.*
from role_menu a
where belong_project = {{.belong_project}}
{{ if .role_id }}
and a.role_id = {{.role_id}}
{{ end  }}