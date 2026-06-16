select
sm.menu_name,
u.nick as create_username,
    CASE
        WHEN a.finish_time IS NULL OR a.create_time IS NULL THEN ''
        WHEN strftime('%s', a.finish_time) < strftime('%s', a.create_time) THEN 'Invalid'
        ELSE time(
            (strftime('%s', a.finish_time) - strftime('%s', a.create_time)),
            'unixepoch'
        )
    END AS duration_hhmmss,
a.*
from (require(base.sql)) as a
left join user_account u on u.user_id = a.create_user
left join sys_menu sm on a.menu_code = sm.menu_code and belong_project= {{.belong_project}}
