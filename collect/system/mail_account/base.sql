select a.*
from mail_account a
where ifnull(a.is_delete, '0') = '0'
{{ if .mail_account_id }}
and a.mail_account_id = {{.mail_account_id}}
{{ end }}
{{ if .email_name }}
and a.email_name = {{.email_name}}
{{ end }}
{{ if .search }}
and a.email_name like {{.search}}
{{ end }}
{{ if .email_name_list }}
and a.email_name in ({{.email_name_list}})
{{ end }}
{{ if .email_name_list_sql }}
and a.email_name in ({{.email_name_list_sql}})
{{ end }}
