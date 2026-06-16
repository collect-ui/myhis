select
tr.sys_code_text as course_type_name,
tp.name as training_project_name,
u.nick as create_username,
a.*
from course_video a
left join user_account u on a.username = u.username
left join sys_code tr on a.course_type = tr.sys_code and tr.sys_code_type='course_type'
left join training_project tp on tp.training_project_id = a.training_project_id
where
1=1
{{ if .course_id }}
and a.course_id = {{.course_id}}
{{ end}}
{{ if .username }}
and a.username = {{.username }}
{{ end}}
order by a.username,a.create_time



