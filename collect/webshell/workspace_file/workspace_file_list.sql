SELECT
  a.*,
  CASE WHEN a.is_dir = '1' THEN '目录' ELSE '文件' END AS file_type_name
FROM (require('./base.sql')) a
ORDER BY
  CASE WHEN a.is_dir = '1' THEN 0 ELSE 1 END ASC,
  CASE WHEN COALESCE(a.order_index, 0) > 0 THEN 0 ELSE 1 END ASC,
  COALESCE(a.order_index, 0) ASC,
  lower(a.name) ASC,
  a.name ASC,
  a.create_time DESC
