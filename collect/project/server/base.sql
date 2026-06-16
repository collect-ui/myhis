select
{{if .with_login_info}}
login_stats.login_count,
{{ end }}
a.*
from server_instance a
{{ if .with_login_info}}
LEFT JOIN (
    -- 获取每个服务器最近半年的登录次数
    SELECT
        sou.server_id,
        COUNT(*) as login_count
    FROM webshell_token wt
    LEFT JOIN server_os_users sou ON sou.server_os_users_id = wt.server_os_users_id
    WHERE sou.server_id IS NOT NULL
        AND wt.create_time >= datetime('now', '-6 months')
    GROUP BY sou.server_id
) login_stats ON login_stats.server_id = a.server_id
{{ end }}

{{ if .project_code }}
left join server_env env on env.server_env_id = a.server_env_id
left join sys_projects p on env.sys_project_id = p.sys_project_id
{{ end }}
where 1=1 and ifnull(a.is_del,'0') = '0'
{{ if .project_code}}
and p.project_code = {{.project_code}}
{{ end }}
{{ if .server_env_id }}
and a.server_env_id = {{.server_env_id}}
{{ end }}
{{ if .search }}
and (
    a.server_ip like {{.search}}
    or a.server_name like {{.search}}
    or exists(
        select 1
        from server_env env
        where env.server_env_id = a.server_env_id
        and ( env.server_env_name like {{.search}} or env.server_env_code like {{.search}} )
    )
)
{{ end }}
{{ if .server_ip }}
and a.server_ip = {{.server_ip}}
{{ end }}
{{ if .server_id }}
and a.server_id = {{.server_id}}
{{ end }}
{{ if .exclude }}
and a.server_id != {{.exclude}}
{{ end }}
{{ if .with_login_info}}
order by login_stats.login_count desc
limit 10
{{ end }}
