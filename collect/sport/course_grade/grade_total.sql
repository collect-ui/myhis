WITH latest_course AS (
    SELECT
        course_id
    FROM
        course
    WHERE
        course_stage IN ('in_finish', 'after_start', 'after_finish')
        and exists(
            select 1
            from course_grade cg
            where cg.course_id = course.course_id
        )
    ORDER BY
        start_time DESC
    LIMIT 1
),
latest_course_scores AS (
    SELECT
        a.username,
        s.nick,
        s.sport_level,
        sl.sys_code_text AS sport_level_name,
        SUM(
            CAST(a.before_score AS DECIMAL(10, 2)) * 0.25 +
            CAST(a.in_score AS DECIMAL(10, 2)) * 0.50 +
            CAST(a.after_score AS DECIMAL(10, 2)) * 0.25
        ) AS total_score
    FROM
        course_grade a
    LEFT JOIN
        user_account s ON a.username = s.username
    LEFT JOIN
        sys_code sl ON s.sport_level = sl.sys_code AND sl.sys_code_type = 'sport_level'
    WHERE
        a.course_id IN (SELECT course_id FROM latest_course) and a.dashboard_type='analyst'
    GROUP BY
        a.username, s.sport_level, s.nick, sl.sys_code_text
),
all_course_counts AS (
    SELECT
        a.username,
        COUNT(*) AS class_count
    FROM
        course_grade a
    where  a.dashboard_type='analyst'
    GROUP BY
        a.username
)
SELECT
    l.username,
    l.sport_level,
    l.total_score,
    l.sport_level_name,
    a.class_count,
    l.nick
FROM
    latest_course_scores l
LEFT JOIN
    all_course_counts a ON l.username = a.username
ORDER BY
    l.total_score DESC, l.username;