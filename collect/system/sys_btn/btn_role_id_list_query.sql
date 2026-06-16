select a.*
from btn_role_id_list a
where 1=1
{{ if  .role_code }}
and a.role_code = {{.role_code}}
{{ end }}
{{ if .menu_code }}
and a.menu_code = {{.menu_code}}
{{ end }}
{{ if and .show_current_user .session_user_id }}
and exists(
 select 1
 from user_role_id_list uril
 where a.role_code  = uril.role_id  and uril.user_id  = {{.session_user_id}}
)
{{ end }}