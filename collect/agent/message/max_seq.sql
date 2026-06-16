select ifnull(max(seq_no), 0) as seq_no
from agent_message
where ifnull(is_delete, '0') = '0'
{{ if .agent_session_id }} and agent_session_id = {{.agent_session_id}} {{ end }}
