select a.*
from feedback_attachment a
where 1=1
{{ if .feedback_id }}
and a.feedback_id = {{.feedback_id}}
{{ end }}