# module=sql

## 作用

- Go 低代码引擎中的 SQL 查询模块。
- 渲染 SQL 模板并执行查询，返回结果列表；可并行执行 `count` 统计。

## 常见用途

- Go 低代码引擎中的 SQL 查询模块。
- 渲染 SQL 模板并执行查询，返回结果列表；可并行执行 `count` 统计。

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | 固定为 `sql` |
| data_file | string | 是 | 主查询 SQL 文件（Go配置常用） |
| count_file | string | 否 | 统计 SQL 文件（Go配置常用） |
| params | object | 否 | 请求参数定义 |
| count_params | object | 否 | 统计参数定义 |
| pagination | bool/string | 否 | 分页开关 |
| count | bool/string | 否 | 统计开关 |
| data_source | string/template | 否 | 指定数据源 |

## 示例

```yml
- key: user_list
  module: sql
  http: true
  params:
    page:
      type: int
      default: 1
    size:
      type: int
      default: 20
    start:
      template: " ({{.page}}-1) * {{.size}}"
      exec: true
      type: int
  data_file: user_list.sql
  count_file: user_list_count.sql
```

## 注意事项

- `count_file` 存在且 `count=true` 时，会并发执行数据SQL和统计SQL。
- 若你的配置使用 `module: mysql`，通常对应 Python 版本实现（参数名是 `sql_file/count_sql`）。

### 源码定位
- Go实现：`/data/project/collect/src/collect/service_imp/module_sql_service.go`

## 元信息

- 来源：`服务文档 -> 模块处理 -> sql / SQL查询执行(Go)`
- 页面标题：`sql(SQL查询执行)`
