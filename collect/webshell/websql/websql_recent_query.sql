SELECT
  recent_sql_id AS id,
  recent_sql_id,
  project_code,
  sql_text AS sql,
  connection_name AS connection,
  connection_id,
  driver,
  statement_type,
  execute_status,
  error_text AS error,
  row_count,
  rows_affected,
  duration_ms,
  execute_count,
  last_executed_at AS executed_at
FROM websql_recent_sql
WHERE project_code = {{.project_code}}
ORDER BY last_executed_at DESC, modify_time DESC
LIMIT {{.size}}
