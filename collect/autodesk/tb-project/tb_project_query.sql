select a.*
from tb_project a
where 1=1
{{ if .name }}
 and a.name = {{.name}}
{{ end }}
{{ if .exclude}}
and a.tb_project_id != {{ .exclude }}
{{ end }}
{{ if .search }}
and  a.name like {{.search}}
{{ end}}
{{ if .with_role_user }}
and ( a.create_user = {{.session_user_id}} or exists(
    select 1
    from user_role_id_list r
    where r.role_id = 'admin' and r.user_id = {{.session_user_id}}
    )
)
{{ end }}
order by a.create_time desc
