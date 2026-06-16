SELECT
    code.sys_code_text as menu_type_name,
    p.menu_name as parent_menu_name,
    a.sys_menu_id,
    a.menu_type,
    a.menu_name,
    a.menu_code,
    a.icon,
    a.is_index,
    a.router_group,
    sm.api as group_api,
    sm.url as group_path,
    a.api,
    a."data",
    a.url,
    a.in_menu,
    a.is_common,
    a.parent_id,
    a.create_time,
    a.create_user,
    a.order_index,
    a.description,
    a.belong_project,
    a."type"
FROM sys_menu a
left join sys_code code on code.sys_code_type = 'menu_type'  and a.menu_type = code.sys_code
left join sys_menu p on a.parent_id = p.sys_menu_id
left join sys_menu sm on a.router_group = sm.menu_code and sm.belong_project = a.belong_project
where a.belong_project= {{.belong_project}}
{{ if .menu_code }}
and a.menu_code = {{.menu_code }}
{{ end }}
{{ if .sys_menu_id}}
and a.sys_menu_id = {{.sys_menu_id }}
{{ end }}

{{ if .is_common}}
and ifnull(a.is_common,'0') = {{.is_common}}
{{ end }}

{{ if .menu_type }}
and a.menu_type = {{.menu_type }}
{{ end }}

{{ if .exclude }}
and a.sys_menu_id != {{.exclude}}
{{ end }}

{{ if .search }}
and ( a.menu_name like {{.search}} or a.url like {{.search}} or a.menu_code like {{.search}})
{{ end }}
{{ if .with_role }}
and (
   1=1
   {{ if .session_user_id }}
    and ifnull(a.is_common,'0')='1' or exists(
        select 1
        from role_menu rm
        join user_role_id_list ur on rm.role_id = ur.role_id and ur.user_id = {{.session_user_id}}
        where rm.sys_menu_id = a.sys_menu_id
   )
   {{ end }}

)
{{end}}


order by a.order_index
