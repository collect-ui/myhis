select distinct f.outp_type_code as "value",
       f.outp_type_code as "label"
from pix_reg_type_fee f
where f.outp_type_code is not null
  and nvl(f.delete_flag, '0') = '1'
order by f.outp_type_code