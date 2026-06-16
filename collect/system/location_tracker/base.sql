
select a.*
from location_tracker a
where 1=1
{{ if .location_tracker_id }}
and a.location_tracker_id = {{.location_tracker_id}}
{{ end}}
{{ if .search }}
and (
a.path like {{.search}} or
a.menu_code like {{.search}}
or a.create_user like {{.search}})
or exists(
    select 1
    from sys_menu sm
    where sm.menu_code = a.menu_code and sm.menu_name like {{.search}}
)
{{ end}}
{{ if .path }}
and a.path = {{.path}}
{{ end }}
{{ if .menu_code }}
and a.menu_code = {{.menu_code}}
{{ end }}
{{ if .create_user }}
and a.create_user = {{.create_user}}
{{ end }}
{{ if .start_time }}
and a.create_time >= {{.start_time}}
{{ end }}
{{ if .end_time }}
and a.create_time <= {{.end_time}}
{{ end }}

{{ if .last_30_min}}
AND create_time >= strftime('%s', 'now', '-30 minutes')
and a.create_user = {{.session_user_id}}
{{end }}

order by a.create_time desc
{{ if .pagination }}
limit {{.start}} , {{.size}}
{{ end }}