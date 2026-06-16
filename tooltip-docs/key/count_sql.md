# count_sql

## 作用

- `count_sql` 指定总数统计 SQL 文件，通常用于分页接口返回 `count`。

## 常见用途

- 列表查询用 `sql_file`，总数查询用 `count_sql`，避免重复写服务。

## 执行阶段（低代码视角）

- 模块执行阶段：执行主查询后或并行执行时，用该 SQL 计算总条数。

## 怎么用

### 配置位置

- `service[].count_sql`（SQL/MYSQL 模块）

## 示例

```yml
service:
  - key: notice_query
    module: sql
    sql_file: notice.sql
    count_sql: count_notice.sql
```

来源文件：`/data/project/moongod-backend/backend_data_service/release/notice/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`count_sql`
