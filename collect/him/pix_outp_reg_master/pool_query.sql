select p.reg_pool_id as "reg_pool_id",
       p.master_id as "master_id",
       p.outp_duration_code as "outp_duration_code",
       p.sort_no as "sort_no",
       p.is_enable as "is_enable",
       p.appointment_flag as "appointment_flag",
       p.reg_flag as "reg_flag",
       p.is_internet as "is_internet"
from pix_outp_reg_pool p
where p.master_id = {{.master_id}}
order by p.sort_no