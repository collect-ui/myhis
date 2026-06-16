select *
from agent_run
where ifnull(is_delete, '0') = '0'
{{ if .agent_run_id }} and agent_run_id = {{.agent_run_id}} {{ end }}
{{ if .agent_session_id }} and agent_session_id = {{.agent_session_id}} {{ end }}
{{ if .request_id }} and request_id = {{.request_id}} {{ end }}
{{ if .status }} and status = {{.status}} {{ end }}
