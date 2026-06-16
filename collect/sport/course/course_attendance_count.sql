
select count(1) course_count
from course a
{{ if .student_id }}
left join course_grade g on a.course_id = g.course_id
{{ end }}
where a.is_delete = '0'
{{ if .training_type }}
and a.training_type = {{.training_type}}
{{ end }}

{{ if .teacher_id }}
and a.teacher_id = {{.teacher_id}}
{{ end }}

{{ if .student_id }}
and g.username = {{.student_id}}
{{ end }}


