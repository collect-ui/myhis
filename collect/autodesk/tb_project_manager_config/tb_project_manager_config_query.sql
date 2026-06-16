select b.nick as project_manager_name,a.*
from tb_project_manager_config a
left join user_account b on a.project_manager = b.username
where 1=1
{{ if .project_manager}}
  and a.project_manager = {{.project_manager}}
{{ end }}
{{ if .exclude }}
and a.tb_project_manager_config_id != {{.exclude}}
{{ end}}
