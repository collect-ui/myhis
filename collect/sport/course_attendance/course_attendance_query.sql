select
tr.sys_code_text as course_type_name,
u.nick as student_name,
tp.name as training_project_name,
tp.img1 as training_type_icon,
a.*
from (require(base.sql)) as a
left join sys_code tr on a.course_type = tr.sys_code and tr.sys_code_type='course_type'
left join user_account u on u.username = a.student_id
left join training_project tp on tp.training_project_id = a.training_project_id
order by a.create_time desc