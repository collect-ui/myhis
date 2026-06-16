SELECT
    schema_page_field_id,
    schema_page_code,
    name,
    key,
    type,
    create_time,
    create_user,
    CAST(order_index AS INTEGER) AS order_index,  -- 将 order_index 转换为数字
    parent_key,
    business_group,
    tag,
    attr,
    link,
    data_attr,
    reg,
    data_from,
    tip,
    value,
    demo_target_row_value
from schema_page_field a
where
{{ if .schema_page_code}}
a.schema_page_code = {{.schema_page_code}}
{{ else}}
a.schema_page_code in ({{.schema_page_code_list}})
{{ end}}
{{ if .search }}
and ( a.name like {{.search}}
    or a.key like {{.search}}
    or a.business_group like {{.search}}
    or a.tag like {{.search}}
    or a.attr like {{.search}}
    or a.link like {{.search}}
    or a.parent_key like {{.search}})
{{ end}}
order by a.order_index+0