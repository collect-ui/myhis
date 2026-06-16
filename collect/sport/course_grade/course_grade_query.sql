select
    c.create_time,
    t.nick as teacher_name,
    s.nick as student_name,
    s.sport_level,
    (
        SELECT GROUP_CONCAT(ur.role_name)
        FROM role ur
        left join user_role_id_list r  on  r.role_id  = ur.role_id
        join role on r.role_id = role.role_id and role.role_code like {{.group_prefix}}
        where  r.user_id  = s.user_id
    ) as group_name,
    a.*,
    (
        CAST(a.before_score AS DECIMAL(10, 2)) * 0.25 +
        CAST(a.in_score AS DECIMAL(10, 2)) * 0.50 +
        CAST(a.after_score AS DECIMAL(10, 2)) * 0.25
    ) as total_score
    {{ if .with_training_project }}
    ,
    tr.training_project_code,
    tr.name as training_project_name
    {{ end }}

from
    course_grade a
left join
    course c on a.course_id = c.course_id
join
    user_account s on a.username = s.username and s.is_delete = '0'
join
    user_account t on c.teacher_id = t.user_id and t.is_delete = '0'
{{ if .with_training_project }}
left join training_project tr on tr.training_project_code = a.training_project_code and tr.is_delete = '0'
{{ end }}
where
{{ if.course_id }}
    a.course_id = {{.course_id}}
{{ else }}
    a.course_id in (
          select c.course_id
          from course_grade b
          join course c on b.course_id =c.course_id
          where b.dashboard_type = 'analyst'
          order by c.start_time desc
          limit 1
    )
{{ end }}
{{ if .username }}
    and a.username = {{ .username }}
{{ end }}
{{ if .dashboard_type }}
and ifnull(a.dashboard_type,'training') = {{.dashboard_type}}
{{ end }}

{{ if .search }}
    and (
        s.nick LIKE {{.search }}
     or s.username LIKE {{.search }}
    )
{{ end}}

{{ if .training_project_code }}
and a.training_project_code = {{.training_project_code}}
{{ end }}

{{ if .with_all }}

union

select
    null as create_time,
    null as teacher_name,
    s.nick as student_name,
    s.sport_level,
    (
            SELECT GROUP_CONCAT(ur.role_name)
            FROM role ur
            left join user_role_id_list r  on  r.role_id  = ur.role_id
            join role on r.role_id = role.role_id and role.role_code like {{.group_prefix}}
            where  r.user_id  = s.user_id
    ) as group_name,
    null as course_grade_id,
    s.username,
    null as before_calories,
    null as before_score,
    null as before_training_times,
    null as before_evaluate,
    null as in_calories,
    null as in_score,
    null as in_training_times,
    null as in_evaluate,
    null as after_calories,
    null as after_score,
    null as after_training_times,
    null as after_evaluate,
    null as course_id,
    null as create_time,
    null as dashboard_type,
    null as training_project_code,
    null as total_score
    {{ if .with_training_project }}
    ,
    tr.training_project_code,
    tr.name as training_project_name
    {{ end }}
from
    user_account s
left join
    sys_code sl on s.sport_level = sl.sys_code and sl.sys_code_type='sport_level'
{{ if .with_training_project }}
left join training_project tr on tr.training_type= {{.training_type}} and tr.is_delete = '0'
    {{ if .body_part}}
    and tr.body_part={{.body_part}}
    {{ end }}
{{ end }}
where
    s.is_delete = '0'
    and not exists (
        select 1
        from course_grade a
        where a.username = s.username
        and a.course_id = {{.course_id}}
        {{ if .dashboard_type }}
            and ifnull(a.dashboard_type,'training') = {{.dashboard_type}}
        {{ end }}
    )
    and not exists(
        select 1
        from user_role_id_list ur
        where s.user_id = ur.user_id and ur.role_id = 'teacher'
    )

{{ end }}

order by
    s.username