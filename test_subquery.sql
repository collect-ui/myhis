select f.outp_type_code,
       f.reg_fee_id,
       f.reg_fee_name,
       (select rf2.reg_fee_name from pix_reg_type_fee rf2
        where rf2.outp_type_code = f.outp_type_code
          and rf2.reg_fee_id = (select min(rf3.reg_fee_id) from pix_reg_type_fee rf3 where rf3.outp_type_code = f.outp_type_code)
          and rownum = 1) as first_name
from pix_reg_type_fee f
where f.reg_fee_id is not null
  and nvl(f.delete_flag, '0') = '1'
  and nvl(f.is_enable, '1') = '1'
  and f.area_code = 'H001'
  and f.outp_type_code = '1'
order by f.reg_fee_id
