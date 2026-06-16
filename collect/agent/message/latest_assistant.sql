select *
from agent_message
where ifnull(is_delete, '0') = '0'
{{ if .agent_run_id }} and agent_run_id = {{.agent_run_id}} {{ end }}
  and role = 'assistant'
order by seq_no desc
limit 1
