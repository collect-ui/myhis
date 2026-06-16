select *
from agent_message
where ifnull(is_delete, '0') = '0'
{{ if .agent_session_id }} and agent_session_id = {{.agent_session_id}} {{ end }}
{{ if .agent_run_id }} and agent_run_id = {{.agent_run_id}} {{ end }}
{{ if .role }} and role = {{.role}} {{ end }}
