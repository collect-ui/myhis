# module=mysql

`module: mysql` 表示当前服务走 SQL 查询执行流程。

## 常见搭配
- `sql_file`
- `count_sql`
- `params` / `count_params`

## 示例
```yml
module: mysql
sql_file: issue_commit_detail.sql
count_sql: count.sql
```
