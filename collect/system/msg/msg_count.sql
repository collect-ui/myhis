
select (
select count(1) as num
from registration
where status = 'pending' and (
      select 1
  from user_role_id_list ur
  join user_account u on ur.user_id = u.user_id
  where ur.role_id in ('admin','project-manage') and u.user_id = {{.session_user_id}}
)) as num,
(
  select  count(1) as feedback_num
  from feedback
  where status = 'start'
   and (
        select 1
    from user_role_id_list ur
    join user_account u on ur.user_id = u.user_id
    where ur.role_id in ('admin','project-manage') and u.user_id = {{.session_user_id}}
  )
) as feedback_num
