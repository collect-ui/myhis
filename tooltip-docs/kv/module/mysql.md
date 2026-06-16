# module=mysql

## 作用

- 使用 SQL 模板文件执行查询，返回列表结果。
- 支持 `count_sql` 统计总数，常用于分页接口。

## 常见用途

- 使用 SQL 模板文件执行查询，返回列表结果。
- 支持 `count_sql` 统计总数，常用于分页接口。

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | 固定为 `mysql` |
| sql_file | string | 是 | 主查询 SQL 文件 |
| count_sql | string | 否 | 统计 SQL 文件 |
| params | object | 否 | 请求参数定义（template/check/default/type） |
| count_params | object | 否 | 统计 SQL 的参数定义 |
| pagination | bool/string | 否 | 是否分页 |
| count | bool/string | 否 | 是否执行总数统计 |
| data_source | string/template | 否 | 指定数据源 |

## 示例

```yml
- key: issue_commit_detail
  module: mysql
  http: true
  params:
    pagination:
      default: true
      type: bool
    page:
      default: 1
      type: int
    size:
      default: 20
      type: int
  sql_file: issue_commit_detail.sql
  count_sql: count.sql
```

## 注意事项

- SQL 模板会先渲染变量，再生成执行参数，支持复杂条件拼接。
- 配置了 `count_sql` 且 `count=true` 时，会执行统计查询。

### 源码定位
- Python类路径：`collect.service_imp.sql.sql_service.SqlService`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/sql/sql_service.py`

## 元信息

- 来源：`服务文档 -> 模块处理 -> mysql / SQL查询执行`
- 页面标题：`mysql(SQL查询执行)`
