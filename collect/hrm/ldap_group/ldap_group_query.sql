select
    {{if .role_id_list }}
     distinct
    {{ end }}
    lg.*
from ldap_group lg
{{if .role_id_list }}
left join role_ldap_group rlg on lg.ldap_group_id = rlg.ldap_group_id
{{end}}
where 1=1
{{ if .search }}
and lg.name like {{.search}}
{{ end}}
{{ if .role_id_list }}
    {{ if .exist}}
    and   rlg.role_id  in ({{.role_id_list }})
    {{ else }}
    and   rlg.role_id  not in ({{.role_id_list }})
    {{end}}
{{ end }}


{{ if .ldap_group_id_list }}
and ldap_group_id in ({{.ldap_group_id_list}})
{{ end }}

{{ if .ldap_group_id }}
and ldap_group_id = {{.ldap_group_id}}
{{ end }}

{{ if .name }}
and name = {{.name}}
{{ end }}
order by lg.order_index desc