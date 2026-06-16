select
tr.sys_code_text as training_type_name,
tr.icon as training_type_icon,
bp.icon as body_part_icon,
bp.sys_code_text as body_part_name,
dl.sys_code_text as difficulty_level_name,
tm.sys_code_text as time_model_name,
a.*
from (require(base.sql)) as a
left join sys_code tr on a.training_type = tr.sys_code and tr.sys_code_type='training_type'
left join sys_code bp on a.body_part = bp.sys_code and bp.sys_code_type='body_part'
left join sys_code dl on a.difficulty_level = dl.sys_code and dl.sys_code_type='difficulty_level'
left join sys_code tm on a.time_model = tm.sys_code and tm.sys_code_type='time_model'
order by a.order_index