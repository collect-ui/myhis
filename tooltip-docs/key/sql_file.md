# sql_file

## 作用

- `sql_file` 指定主查询 SQL 文件路径（相对当前服务目录）。

## 常见用途

- `module: sql/mysql` 的查询服务把 SQL 外置到独立文件维护。

## 执行阶段（低代码视角）

- 模块执行阶段：读取并渲染 SQL 后执行数据库查询。

## 怎么用

### 配置位置

- `service[].sql_file`

## 示例

```yml
service:
  - key: notice_query
    module: sql
    sql_file: notice.sql
```

来源文件：`/data/project/moongod-backend/backend_data_service/release/notice/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`sql_file`
