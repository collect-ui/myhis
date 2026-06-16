select f.reg_fee_id,
       f.reg_fee_name,
       f.outp_type_code,
       f.area_code
from pix_reg_type_fee f
where f.reg_fee_id is not null
  and nvl(f.delete_flag, '0') = '1'
  and nvl(f.is_enable, '1') = '1'
  and f.reg_fee_id = {{.reg_fee_id}}
fetch first 1 rows only
