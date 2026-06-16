BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS mail_account (
  mail_account_id VARCHAR(50) PRIMARY KEY,
  order_index INT,
  email_name VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  guid_code VARCHAR(255) NOT NULL,
  recovery_code TEXT NOT NULL,
  raw_text TEXT NOT NULL,
  is_current_running VARCHAR(10) DEFAULT '0',
  current_run_mark_time VARCHAR(255) DEFAULT '',
  proton_registered VARCHAR(10) DEFAULT '0',
  proton_email VARCHAR(255) DEFAULT '',
  proton_password VARCHAR(255) DEFAULT 'Zhangzhi@888',
  codex_device_code TEXT DEFAULT '',
  codex_device_auth_id TEXT DEFAULT '',
  codex_user_code TEXT DEFAULT '',
  codex_authorization_code TEXT DEFAULT '',
  codex_code_verifier TEXT DEFAULT '',
  codex_verification_uri TEXT DEFAULT '',
  codex_verification_uri_complete TEXT DEFAULT '',
  codex_interval VARCHAR(50) DEFAULT '',
  codex_expires_in VARCHAR(50) DEFAULT '',
  codex_access_token TEXT DEFAULT '',
  codex_refresh_token TEXT DEFAULT '',
  codex_id_token TEXT DEFAULT '',
  codex_token_type VARCHAR(100) DEFAULT '',
  codex_scope TEXT DEFAULT '',
  codex_expires_at VARCHAR(255) DEFAULT '',
  codex_account_id VARCHAR(255) DEFAULT '',
  codex_usage_json TEXT DEFAULT '',
  codex_usage_plan_type VARCHAR(100) DEFAULT '',
  codex_usage_allowed VARCHAR(20) DEFAULT '',
  codex_usage_limit_reached VARCHAR(20) DEFAULT '',
  codex_usage_used_percent VARCHAR(50) DEFAULT '',
  codex_usage_reset_at VARCHAR(255) DEFAULT '',
  codex_usage_last_query_time VARCHAR(255) DEFAULT '',
  codex_usage_msg TEXT DEFAULT '',
  codex_auth_json TEXT DEFAULT '',
  codex_auth_status VARCHAR(100) DEFAULT '',
  codex_auth_msg TEXT DEFAULT '',
  codex_last_auth_time VARCHAR(255) DEFAULT '',
  create_time VARCHAR(255),
  create_user VARCHAR(255),
  is_delete VARCHAR(10) DEFAULT '0'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mail_account_email_name
ON mail_account(email_name);

CREATE INDEX IF NOT EXISTS idx_mail_account_create_time
ON mail_account(create_time);

DELETE FROM role_menu
WHERE sys_menu_id = '9f4f5b90-2ee6-49ac-94dd-f5d8516bb809';

DELETE FROM sys_menu
WHERE menu_code = 'mail_account'
  AND belong_project = 'base';

INSERT INTO sys_menu (
  sys_menu_id,
  menu_type,
  menu_name,
  menu_code,
  icon,
  is_index,
  group_path,
  router_group,
  group_api,
  api,
  data,
  url,
  in_menu,
  is_common,
  parent_id,
  create_time,
  create_user,
  order_index,
  description,
  belong_project,
  type
)
SELECT
  '9f4f5b90-2ee6-49ac-94dd-f5d8516bb809',
  '2',
  '邮箱登记',
  'mail_account',
  'FaEnvelope',
  '0',
  '',
  'framework',
  '',
  'post:/template_data/data?service=frontend.mail_account',
  '',
  '/framework/mail_account',
  '1',
  '1',
  '7c8b9620-db64-4586-97f6-a715c6d477b7',
  datetime('now', 'localtime'),
  'seed',
  2,
  '邮箱批量导入与查询页面',
  'base',
  ''
WHERE EXISTS (
  SELECT 1
  FROM sys_menu
  WHERE sys_menu_id = '7c8b9620-db64-4586-97f6-a715c6d477b7'
    AND belong_project = 'base'
);

COMMIT;
