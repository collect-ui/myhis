select
distinct
a.parent_name,a.name
from doc_project_detail a
left join doc_project p on a.doc_project_id = p.doc_project_id
left join tb_project p2 on p.tb_project_id = p2.tb_project_id
where 1=1
and p2.tb_project_id = {{.tb_project_id}}