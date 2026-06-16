select e.doctor_code as "value",
       nvl(u.people_name, e.doctor_code) || '（' || e.doctor_code || '）' as "label",
       e.doctor_code as "doctor_code",
       u.people_name as "doctor_name",
       e.area_code as "area_code",
       e.reside_dept_code as "reside_dept_code",
       e.special_clinic_code as "special_clinic_code",
       s.special_clinic_name as "special_clinic_name",
       e.average_time as "average_time"
from pix_outp_emp_consulting_room e
left join pix_outp_special_clinic s
  on s.special_clinic_code = e.special_clinic_code
 and nvl(s.delete_flag, '1') = '1'
left join bcs.bcs_user u
  on u.user_name = e.doctor_code
where e.doctor_code is not null
  and nvl(e.is_enable, '1') = '1'
  and u.status = 'A'
  {{if .area_code}}and e.area_code = {{.area_code}}{{end}}
  {{if .reside_dept_code}}and e.reside_dept_code = {{.reside_dept_code}}{{end}}
  {{if .special_clinic_code}}and e.special_clinic_code = {{.special_clinic_code}}{{end}}
  {{if .doctor_code}}and e.doctor_code = {{.doctor_code}}{{end}}
  {{if .keyword}}and (u.people_name like {{.keyword}} or e.doctor_code like {{.keyword}} or u.py_code like {{.keyword}} or u.wb_code like {{.keyword}}){{end}}
  {{if not .special_clinic_code}}and not exists (
    select 1 from pix_outp_emp_consulting_room e2
    where e2.doctor_code = e.doctor_code
      and nvl(e2.is_enable, '1') = '1'
      and e2.special_clinic_code > e.special_clinic_code
      {{if .area_code}}and e2.area_code = {{.area_code}}{{end}}
      {{if .reside_dept_code}}and e2.reside_dept_code = {{.reside_dept_code}}{{end}}
  ){{end}}
order by e.outp_empconsult_id desc
