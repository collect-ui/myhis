select *
from agent_run
where ifnull(is_delete, '0') = '0'
  and status = 'running'
  and lease_expire_time <> ''
  and lease_expire_time < {{.lease_expire_before}}
order by lease_expire_time asc
limit 20
