WITH student_scores AS (
    SELECT
        s.nick ,
        t.nick as teacher_name,
        s.sport_level,
        sl.sys_code_text AS sport_level_name,
        (
            CAST(a.before_score AS DECIMAL(10, 2)) * 0.25 +
            CAST(a.in_score AS DECIMAL(10, 2))  * 0.50 +
            CAST(a.after_score AS DECIMAL(10, 2)) * 0.25
        ) AS total_score,
        c.course_stage,
        c.start_time,
        c.teacher_id,
        tr.sys_code_text as training_type_name,
        a.*
    FROM
        course_grade a
    LEFT JOIN
        user_account s ON a.username = s.username
    left join course c on a.course_id = c.course_id
    left join user_account t on c.teacher_id = t.user_id
    left join sys_code tr on tr.sys_code = c.training_type and tr.sys_code_type = 'training_type'
    LEFT JOIN
        sys_code sl ON s.sport_level = sl.sys_code AND sl.sys_code_type = 'sport_level'
    where  1=1 and c.course_stage != 'not_start'
    {{ if .teacher_id }}
       and c.teacher_id = {{.teacher_id}}
    {{ end }}
    {{ if .student_id }}
        and a.username = {{.student_id}}
    {{ end }}
)
SELECT
    s.*
FROM
    student_scores s
ORDER BY
    s.start_time desc,s.total_score desc