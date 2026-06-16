# count_params

## 作用

- `count_params` 用于给 `count_sql` 提供独立参数，常用于“总数查询与列表查询参数差异化”。

## 常见用途

- 分页列表查询里关闭 `pagination`，避免总数 SQL 被分页参数干扰。

## 执行阶段（低代码视角）

- 模块执行阶段（SQL 总数查询前）：先组装 `count_params`，再执行 `count_sql`。

## 怎么用

### 配置位置

- `service[].count_params`（搭配 `count_sql`）

## 示例

```yml
service:
  - key: notice_query
    module: sql
    sql_file: notice.sql
    count_sql: count_notice.sql
    count_params:
      pagination:
        default: false
        type: bool
```

来源文件：`/data/project/moongod-backend/backend_data_service/release/notice/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`count_params`
