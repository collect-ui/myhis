select f.reg_fee_id as "value",
       f.reg_fee_name as "label",
       f.reg_fee_id as "reg_fee_id",
       f.reg_fee_name as "reg_fee_name",
       f.outp_type_code as "outp_type_code",
       f.reg_fee_name as "outp_type_code_str",
       f.area_code as "area_code",
       f.py_code as "py_code",
       f.wb_code as "wb_code",
       f.reserve_source as "reserve_source"
from pix_reg_type_fee f
where f.reg_fee_id is not null
  and nvl(f.delete_flag, '0') = '1'
  and nvl(f.is_enable, '1') = '1'
  {{if .area_code}}and f.area_code = {{.area_code}}{{end}}
  {{if .reg_fee_id}}and f.reg_fee_id = {{.reg_fee_id}}{{end}}