UPDATE webshell_quick_text
SET
  use_count = COALESCE(use_count, 0) + 1,
  modify_time = {{.now_time}},
  modify_user = {{if .session_user_id}}{{.session_user_id}}{{else}}''{{end}}
WHERE quick_text_id = {{.quick_text_id}}
  AND COALESCE(is_delete, '0') = '0'
RETURNING quick_text_id, use_count;
