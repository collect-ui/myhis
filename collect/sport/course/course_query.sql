select
tr.sys_code_text as training_type_name,
tr.icon as training_type_icon,
u.nick as teacher_name,
cs.sys_code_text as course_stage_name,
bp.sys_code_text as body_part_name,
(
    select count(1)
    from course_attendance ca
    where ca.course_id = a.course_id
) as attendance_count,
a.*
from (require(base.sql)) as a
left join sys_code tr on a.training_type = tr.sys_code and tr.sys_code_type='training_type'
left join sys_code cs on a.course_stage = cs.sys_code and cs.sys_code_type='course_stage'
left join sys_code bp on a.body_part = bp.sys_code and bp.sys_code_type='body_part'
left join user_account u on u.user_id = a.teacher_id
order by a.create_time desc