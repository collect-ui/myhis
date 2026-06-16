DELETE FROM sys_menu
WHERE menu_code = 'agent_regression'
  AND belong_project = 'base';

INSERT INTO sys_menu (
  sys_menu_id, menu_type, menu_name, menu_code, icon, is_index,
  group_path, router_group, group_api, api, data, url,
  in_menu, is_common, parent_id, create_time, create_user,
  order_index, description, belong_project, type
) VALUES (
  '4dcfec14-3e9d-4baa-8094-17f9fa8d2a4d',
  '2',
  'Agent测试回归',
  'agent_regression',
  '',
  '0',
  '',
  'framework',
  '',
  'post:/template_data/data?service=frontend.agent_regression',
  '',
  '/framework/agent_regression',
  '1',
  '1',
  '7c8b9620-db64-4586-97f6-a715c6d477b7',
  '2026-04-23 09:35:00',
  '',
  3,
  'Agent 测试回归页面入口',
  'base',
  ''
);
