select
  ('g_' || g.doc_group_id) as id,
  '0' as parent_id,
  g.name as title,
  ifnull(g."desc", '') as cn_title,
  case
    when ifnull(g."desc", '') = '' then g.name
    else (g.name || '(' || g."desc" || ')')
  end as display_title,
  '1' as is_dir,
  g.type as node_type,
  g.doc_group_id as doc_group_id,
  '' as collect_doc_id,
  g.order_index as order_index
from doc_group g
where g.is_delete = '0'
  and lower(ifnull(g.type, '')) in ('2', 'service')
{{ if .project_code }}
  and ifnull(g.project_code, '') = {{.project_code}}
{{ end }}

union all

select
  ('d_' || d.collect_doc_id) as id,
  ('g_' || d.parent_dir) as parent_id,
  d.title as title,
  ifnull(d.sub_title, '') as cn_title,
  case
    when ifnull(d.sub_title, '') = '' then d.title
    else (d.title || '(' || d.sub_title || ')')
  end as display_title,
  '0' as is_dir,
  d.type as node_type,
  d.parent_dir as doc_group_id,
  d.collect_doc_id as collect_doc_id,
  d.order_index as order_index
from collect_doc d
join doc_group g on d.parent_dir = g.doc_group_id
  and g.is_delete = '0'
  and lower(ifnull(g.type, '')) in ('2', 'service')
where d.is_delete = '0'
{{ if .project_code }}
  and ifnull(d.project_code, '') = {{.project_code}}
  and ifnull(g.project_code, '') = {{.project_code}}
{{ end }}

order by is_dir desc, order_index asc
