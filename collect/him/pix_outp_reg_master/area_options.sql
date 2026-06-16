select a.area_code as "value",
       nvl(a.area_name, a.area_code) as "label",
       a.area_code as "area_code",
       nvl(a.area_name, a.area_code) as "area_name"
from bcs.bcs_hospital_area a
where a.area_code is not null
order by a.area_code
