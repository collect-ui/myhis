# module=bulk_service

## 作用

- 按 `foreach` 批量触发子服务，支持并发执行。
- 用于把一批输入项分发给同一个服务模板处理。

## 常见用途

- 按 `foreach` 批量触发子服务，支持并发执行。
- 用于把一批输入项分发给同一个服务模板处理。

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | 固定为 `bulk_service` |
| batch.foreach | string | 是 | 循环数据字段名 |
| batch.item | string | 是 | 当前项变量名 |
| batch.service | object | 是 | 子服务定义（含 `service: xxx.xxx`） |
| batch.max_once | int | 否 | 单批并发数，默认 30 |
| batch.append_param | bool | 否 | 是否透传主参数到子服务，默认 true |
| batch.append_item_param | bool | 否 | 是否把当前项展开到参数中 |
| batch.save_field | string | 否 | 汇总结果保存字段 |

## 示例

```yml
- key: collect_doc_batch_query
  module: bulk_service
  handler_params:
    - key: service2field
      service:
        service: config.get_detail_service
        collect_doc_id: "[collect_doc_id]"
      save_field: service_list
  batch:
    foreach: "[service_list]"
    item: item
    service:
      service: "[service]"
    append_item_param: true
    save_field: result
```

## 注意事项

- 并发结果会按输入顺序汇总，并附带原始 `item`。
- `foreach` 为空时返回空结果，不报错。
- 内部限制一次最多处理 100000 条。

### 源码定位
- Python类路径：`collect.service_imp.bulk.bulk_service.ServiceBulkService`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/bulk/bulk_service.py`

## 元信息

- 来源：`服务文档 -> 模块处理 -> bulk_service / 批量服务执行`
- 页面标题：`bulk_service(批量服务执行)`
