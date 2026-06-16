select o.org_code as "value",
       o.org_name || '（' || o.org_code || '）' as "label",
       o.org_code as "org_code",
       o.org_name as "org_name",
       o.area_code as "area_code",
       nvl(o.is_emergency, '0') as "is_emergency",
       o.py_code as "py_code",
       o.wb_code as "wb_code"
from bcs.bcs_org o
where o.org_code is not null
  and nvl(o.delete_flag, '1') = '1'
  and nvl(o.is_enable, '1') = '1'
  and o.clic_detail_type = '3'
  {{if .area_code}}and o.area_code = {{.area_code}}{{end}}
  {{if .reside_dept_code}}and o.org_code = {{.reside_dept_code}}{{end}}
order by o.org_id
