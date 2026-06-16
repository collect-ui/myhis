select a.*,
(
    select count(1)
    from user_role_id_list b
    join user_account c on c.user_id = b.user_id and c.is_delete= '0'
    where b.role_id = a.role_id
) as user_count
from (require(base.sql)) as a
order by a.order_index