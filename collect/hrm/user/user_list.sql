select


    a.user_id,
    {{ if eq (get_key "system_model") "company" }}
    a.userpwd as password,
    {{else }}
    a.password as password,
    {{ end }}
    (
        SELECT GROUP_CONCAT(ur.role_name)
        FROM role ur
        left join user_role_id_list r  on  r.role_id  = ur.role_id
        where  r.user_id  = a.user_id
    ) as role_names,
    (
        SELECT GROUP_CONCAT(ur.role_code)
        FROM role ur
        left join user_role_id_list r  on  r.role_id  = ur.role_id
        where  r.user_id  = a.user_id
    ) as roles,
    c.sys_code_text as status_text,
    CASE
        WHEN EXISTS (
            SELECT 1
            FROM user_role_id_list ur
            WHERE a.user_id = ur.user_id AND ur.role_id = 'teacher'
        ) THEN '1'
        ELSE '0'
    END as has_teacher,
    sl.sys_code_text as sport_level_name,
    sex.sys_code_text as sex_name,
    a.*
from user_account a
left join sys_code c on a.status = c.sys_code and  c.sys_code_type = 'user_job_status'
left join sys_code sl on a.sport_level = sl.sys_code and sl.sys_code_type='sport_level'
left join sys_code sex on a.sex = sex.sys_code and sex.sys_code_type='sex'
where
ifnull(a.status,'1')!='0' and a.is_delete = '0'
require('./base_where.common')



