select b.link,a.key,a.value,b.order_index
from tb_config a
join schema_page_field b on b.schema_page_code = {{.schema_page_code}}
and b.tag='attachment' and a.key = b.key and ifnull(b.link,'') != ''
where a.tb_project_id ={{.tb_project_id}} and a.value != ''