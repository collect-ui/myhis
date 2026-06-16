SELECT
  a.*, 
  CASE
    WHEN si.server_busi_name IS NOT NULL AND si.server_busi_name != ''
      THEN (si.server_busi_name || '-' || si.server_ip)
    ELSE si.server_ip
  END AS server_name,
  sou.user_name AS server_user_name
FROM (require('./base.sql')) a
LEFT JOIN server_instance si ON si.server_id = a.server_id
LEFT JOIN server_os_users sou ON sou.server_os_users_id = a.server_os_users_id
ORDER BY a.order_id ASC, a.create_time DESC
