select reg_pool_id, master_id, sort_no, appointment_flag, is_internet, is_enable, reg_flag, outp_duration_code
from pix_outp_reg_pool
where master_id = {{.master_id}}
order by sort_no
