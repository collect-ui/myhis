-- 挂号类别代码表 PIX0018（对应老系统 BCS.BCS_CODE_TABLE + BCS_CODE_TABLE_ITEM）
-- 用于 outp_type_code 翻译

MERGE INTO sys_code t
USING (
  SELECT sys_guid() AS sys_code_id, 'outp_type_code' AS sys_code_type, '挂号类别' AS sys_code_type_name,
         code_val AS sys_code, code_name AS sys_code_text, sort_no AS order_index
  FROM (
    SELECT '1'  AS code_val, '普通号'    AS code_name, 1  AS sort_no FROM dual UNION ALL
    SELECT '2',  '急诊',      2  FROM dual UNION ALL
    SELECT '3',  '专家号',    3  FROM dual UNION ALL
    SELECT '9',  '义诊',      4  FROM dual UNION ALL
    SELECT '12', '中医专家',  5  FROM dual UNION ALL
    SELECT '14', '免费',      6  FROM dual UNION ALL
    SELECT '15', '知名专家',  7  FROM dual UNION ALL
    SELECT '21', '儿科专家',  8  FROM dual UNION ALL
    SELECT '24', '中医普通',  9  FROM dual UNION ALL
    SELECT '31', '午间急诊', 10  FROM dual UNION ALL
    SELECT '32', '晚间急诊', 11  FROM dual UNION ALL
    SELECT '33', '凌晨急诊', 12  FROM dual
  )
) s
ON (t.sys_code_type = s.sys_code_type AND t.sys_code = s.sys_code)
WHEN NOT MATCHED THEN
  INSERT (sys_code_id, sys_code_type, sys_code_type_name, sys_code, sys_code_text, order_index)
  VALUES (s.sys_code_id, s.sys_code_type, s.sys_code_type_name, s.sys_code, s.sys_code_text, s.order_index)
WHEN MATCHED THEN
  UPDATE SET t.sys_code_text = s.sys_code_text, t.order_index = s.order_index
