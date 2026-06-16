select s.schedule_id as "schedule_id",
       s.master_id as "master_id",
       s.schedule_date as "schedule_date",
       s.outp_duration_code as "outp_duration_code",
       s.doctor_code as "doctor_code",
       s.registration_limits as "registration_limits",
       s.appointment_limits as "appointment_limits",
       s.internet_limits as "internet_limits",
       s.atime_flag as "atime_flag",
       s.is_enable as "is_enable",
       s.delete_flag as "delete_flag",
       s.schedule_status as "schedule_status"
from pix_outp_reg_schedule s
where s.master_id = {{.master_id}}