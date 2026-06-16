select s.special_clinic_code as "value",
       s.special_clinic_name || '（' || s.special_clinic_code || '）' as "label",
       s.special_clinic_code as "special_clinic_code",
       s.special_clinic_name as "special_clinic_name",
       s.reside_dept_code as "reside_dept_code",
       s.area_code as "area_code",
       s.schedule_flag as "schedule_flag",
        s.py_code as "py_code",
        s.wb_code as "wb_code",
        s.outp_duration_codes as "outp_duration_codes"
from pix_outp_special_clinic s
where s.special_clinic_code is not null
  and nvl(s.delete_flag, '1') = '1'
  and nvl(s.is_enable, '1') = '1'
  and nvl(s.schedule_flag, '0') = '1'
  {{if .area_code}}and s.area_code = {{.area_code}}{{end}}
  {{if .reside_dept_code}}and s.reside_dept_code = {{.reside_dept_code}}{{end}}
  {{if .special_clinic_code}}and s.special_clinic_code = {{.special_clinic_code}}{{end}}
order by s.special_clinic_code
