select a.*
from work_task_project a
where a.is_delete = '0'
order by a.order_index