select *
from agent_run
where ifnull(is_delete, '0') = '0'
  and status = 'queued'
order by create_time asc
limit 5
