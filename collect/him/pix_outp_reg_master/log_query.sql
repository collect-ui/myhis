select l.master_log_id as "master_log_id",
       l.master_id as "master_id",
       l.log_date as "log_date",
       l.operator as "operator",
       l.log_text as "log_text"
from pix_outp_reg_master_log l
where l.master_id = {{.master_id}}