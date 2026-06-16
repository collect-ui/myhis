select a.*
from course a
left join user_account u on u.user_id = a.teacher_id
where a.is_delete = '0'
{{ if .course_id}}
and a.course_id = {{.course_id }}
{{ end }}
{{ if .teacher_id }}
and a.teacher_id like {{.teacher_id}}
{{ end}}
{{  if .search }}
and (u.nick like {{.search}} or u.username like {{.search}})
{{ end }}
{{ if .start_time}}
and a.start_time >= {{.start_time }}
{{ end }}
{{ if .end_time}}
and a.start_time <= {{.end_time }}
{{ end }}
{{ if .is_start }}
and a.course_stage !='not_start'
{{ end }}
{{ if .training_type }}
and a.training_type = {{.training_type}}
{{ end }}
{{ if .has_grade }}
and exists (
       select 1
    from course_grade
    where course_grade.course_id = a.course_id
    {{ if .dashboard_type }}
        and course_grade.dashboard_type = {{.dashboard_type}}
    {{ end }}
    limit 1
)
{{ end}}

order by a.create_time desc
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}