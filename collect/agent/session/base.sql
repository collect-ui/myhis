select *
from agent_session
where ifnull(is_delete, '0') = '0'
{{ if .agent_session_id }} and agent_session_id = {{.agent_session_id}} {{ end }}
{{ if .session_key }} and session_key = {{.session_key}} {{ end }}
{{ if .scene_code }} and scene_code = {{.scene_code}} {{ end }}
{{ if .status }} and status = {{.status}} {{ end }}
