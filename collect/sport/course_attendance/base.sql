select a.*
from course_attendance a
{{ if .search }}
left join user_account u on u.user_id = a.student_id
{{ end }}
where 1=1
{{ if .course_id }}
and a.course_id like {{.course_id}}
{{ end}}
{{ if .search }}
and (
    u.username like {{.search}} or u.nick like {{.search}}
)
{{ end}}


{{  if .student_id }}
and a.student_id = {{.student_id }}
{{ end }}

order by a.create_time desc
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}