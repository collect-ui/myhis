-- SQLite migration for the Windows price.db deployment.
-- Scope: create the WebSQL runtime tables and indexes that exist locally but
-- are missing on the Windows host.
-- Deliberately omitted: webshell_workspace_project column-order-only diff.

BEGIN IMMEDIATE;

CREATE TABLE IF NOT EXISTS `websql_commit_event` (
  `websql_commit_event_id` text,
  `status` text,
  `commit_mode` text,
  `driver` text,
  `database_name` text,
  `connection_id` text,
  `connection_name` text,
  `statement_type` text,
  `sql_text` text,
  `rows_affected` integer,
  `last_insert_id` integer,
  `create_time` text,
  `expire_time` text,
  `finish_time` text,
  `duration_ms` integer,
  `error_text` text,
  PRIMARY KEY (`websql_commit_event_id`)
);

CREATE TABLE IF NOT EXISTS `websql_completion_probe` (
  `id` INTEGER PRIMARY KEY,
  `name` TEXT,
  `completion_note` TEXT
);

CREATE TABLE IF NOT EXISTS `websql_favorite` (
  `websql_favorite_id` text,
  `project_code` text,
  `item_type` text,
  `name` text,
  `path` text,
  `folder` text,
  `sql_text` text,
  `connection_id` text,
  `connection_name` text,
  `driver` text,
  `is_delete` text,
  `create_time` text,
  `create_user` text,
  `modify_time` text,
  `modify_user` text,
  PRIMARY KEY (`websql_favorite_id`)
);

CREATE TABLE IF NOT EXISTS `websql_recent_sql` (
  `recent_sql_id` text,
  `project_code` text,
  `recent_sql_hash` text,
  `driver` text,
  `connection_id` text,
  `connection_name` text,
  `statement_type` text,
  `sql_text` text,
  `execute_status` text,
  `error_text` text,
  `row_count` integer,
  `rows_affected` integer,
  `duration_ms` integer,
  `execute_count` integer,
  `last_executed_at` text,
  `create_time` text,
  `create_user` text,
  `modify_time` text,
  `modify_user` text,
  PRIMARY KEY (`recent_sql_id`)
);

CREATE INDEX IF NOT EXISTS `idx_websql_favorite_folder`
  ON `websql_favorite` (`folder`);
CREATE INDEX IF NOT EXISTS `idx_websql_favorite_is_delete`
  ON `websql_favorite` (`is_delete`);
CREATE INDEX IF NOT EXISTS `idx_websql_favorite_item_type`
  ON `websql_favorite` (`item_type`);
CREATE INDEX IF NOT EXISTS `idx_websql_favorite_path`
  ON `websql_favorite` (`path`);
CREATE INDEX IF NOT EXISTS `idx_websql_favorite_project_code`
  ON `websql_favorite` (`project_code`);

CREATE INDEX IF NOT EXISTS `idx_websql_recent_sql_connection_id`
  ON `websql_recent_sql` (`connection_id`);
CREATE INDEX IF NOT EXISTS `idx_websql_recent_sql_driver`
  ON `websql_recent_sql` (`driver`);
CREATE INDEX IF NOT EXISTS `idx_websql_recent_sql_last_executed_at`
  ON `websql_recent_sql` (`last_executed_at`);
CREATE INDEX IF NOT EXISTS `idx_websql_recent_sql_project_code`
  ON `websql_recent_sql` (`project_code`);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_websql_recent_sql_recent_sql_hash`
  ON `websql_recent_sql` (`recent_sql_hash`);

COMMIT;
