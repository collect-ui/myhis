# module=bulk_upsert

## 作用

- 批量修改多行记录
- 批量修改多行记录，支持修改不同记录不同值
- 像excel 导入可以用此模块，或者前台传大量数据对象过来
- 一条一条更新数据库连接释放比较慢，这个是一次性更新

## 常见用途

- 批量修改多行记录
- 批量修改多行记录，支持修改不同记录不同值

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | bulk_upsert |
| table | string | 是 | 修改哪个表 |
| model_field | string | 是 | 数据列表取哪个参数,[你的参数] |
| update_fields | array |  | 更新哪些字段，不填所有字段 |
| options | string |  | 支持update_fields 取动态变量，[你参数变量] |
| ignore_fields | array |  | 忽略哪些字段 |

## 示例

```yml
- key: collect_doc_params_update
  module: bulk_upsert
  table: "collect_doc_params"
  params:
    params:
      check:
        template: "{{must .params}}"
        err_msg: 参数不能为空
  model_field: "[params]"
```

## 注意事项

- 批量修改多行记录，支持修改不同记录不同值。像excel 导入可以用此模块，或者前台传大量数据对象过来。一条一条更新数据库连接释放比较慢，这个是一次性更新

## 元信息

- 来源：`服务文档 -> 模块处理 -> 7.bulk_upsert / 批量修改多行`
- 页面标题：`bulk_upsert(批量修改多行)`
