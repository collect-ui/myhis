select a.master_id as "master_id",
       a.area_code as "area_code",
       a.outp_date as "outp_date",
       a.outp_special_id as "outp_special_id",
       a.reside_dept_code as "reside_dept_code",
       a.special_clinic_code as "special_clinic_code",
       a.outp_title_code as "outp_title_code",
       a.doctor_code as "doctor_code",
       a.outp_type_code as "outp_type_code",
       a.outp_duration_code as "outp_duration_code",
       a.registration_limits as "registration_limits",
       a.appointment_limits as "appointment_limits",
       a.current_limits as "current_limits",
       a.appointment_current_limits as "appointment_current_limits",
       a.atime_flag as "atime_flag",
       a.is_enable as "is_enable",
       a.upload_flag as "upload_flag",
       a.modify_flag as "modify_flag",
       a.current_no as "current_no",
       a.sort_no as "sort_no",
       a.transfer_no as "transfer_no",
       a.alias_name as "alias_name",
       a.master_status as "master_status",
       a.descn as "descn",
       a.old_doctor_code as "old_doctor_code",
       a.reg_fee_id as "reg_fee_id",
       a.registration_type as "registration_type",
       a.internet_limits as "internet_limits",
       a.average_time as "average_time"
from pix_outp_reg_master a
where 1=1
{{ if .master_id }}
and a.master_id = {{.master_id}}
{{ end }}
{{ if .area_code }}
and a.area_code like {{.area_code}}
{{ end }}
{{ if .doctor_code }}
and a.doctor_code like {{.doctor_code}}
{{ end }}
{{ if .outp_date }}
and a.outp_date = {{.outp_date}}
{{ end }}
{{ if .is_enable }}
and a.is_enable = {{.is_enable}}
{{ end }}
{{ if .master_status }}
and a.master_status = {{.master_status}}
{{ end }}
order by a.outp_date desc, a.sort_no asc