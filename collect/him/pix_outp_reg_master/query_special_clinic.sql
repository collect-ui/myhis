select s.special_clinic_code,
       s.special_clinic_name,
       s.outp_special_id,
       s.reside_dept_code,
       s.area_code
from pix_outp_special_clinic s
where s.special_clinic_code is not null
  and nvl(s.delete_flag, '1') = '1'
  and nvl(s.is_enable, '1') = '1'
  and s.special_clinic_code = {{.special_clinic_code}}
fetch first 1 rows only
