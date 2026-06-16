select
  0 as "reg_pool_id",
  {{.master_id}} as "master_id",
  {{.outp_duration_code}} as "outp_duration_code",
  level as "sort_no",
  '1' as "is_enable",
  '1' as "appointment_flag",
  '0' as "is_internet",
  '0' as "reg_flag"
from dual
connect by level <= {{.appointment_limits}}
union all
select
  0 as "reg_pool_id",
  {{.master_id}} as "master_id",
  {{.outp_duration_code}} as "outp_duration_code",
  {{.appointment_limits}} + level as "sort_no",
  '1' as "is_enable",
  '0' as "appointment_flag",
  '0' as "is_internet",
  '0' as "reg_flag"
from dual
connect by level <= {{.no_appointment_count}}
union all
select
  0 as "reg_pool_id",
  {{.master_id}} as "master_id",
  {{.outp_duration_code}} as "outp_duration_code",
  {{.appointment_limits}} + {{.no_appointment_count}} + level as "sort_no",
  '1' as "is_enable",
  '1' as "appointment_flag",
  '1' as "is_internet",
  '0' as "reg_flag"
from dual
connect by level <= {{.internet_limits}}