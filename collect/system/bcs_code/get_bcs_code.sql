SELECT
  C.STANDARD_CODE AS STANDARD_CODE,
  T.ITEM_VALUE AS ITEM_VALUE,
  T.ITEM_NAME AS ITEM_NAME
FROM
  BCS.BCS_CODE_TABLE_ITEM T
  LEFT JOIN BCS.BCS_CODE_TABLE C ON C.TYPE_ID = T.T_ID
WHERE
  T.AUDIT_STATUS = '2'
  AND T.IS_ENABLE = '1'
  {{if .standard_code}}AND C.STANDARD_CODE = {{.standard_code}}{{end}}
  {{if .standard_code_list}}AND C.STANDARD_CODE IN ({{.standard_code_list}}){{end}}
  {{if .area_code}}AND (T.AREA_CODE IS NULL OR T.AREA_CODE = {{.area_code}}){{end}}
  {{if .sys_code_list}}AND T.ITEM_VALUE IN ({{.sys_code_list}}){{end}}
