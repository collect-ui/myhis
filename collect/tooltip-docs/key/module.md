# module

服务主体模块，决定当前服务由哪类执行器处理。

## 作用
- 这是服务级核心关键字。
- 不同 `module` 会触发不同实现逻辑。

## 常见取值
- `mysql`：SQL 查询模块
- `http`：HTTP 转发模块
- `empty`：空模块（通常配合 handler_params/result_handler）
- `bulk_create`：批量新增模块

## 示例
```yml
module: mysql
sql_file: issue_commit_detail.sql
```
