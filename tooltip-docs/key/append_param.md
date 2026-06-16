# append_param

## 作用

- `append_param` 用于控制调用子服务时，是否把当前参数上下文一起透传给子服务。
- 常见于 `service2field`、`combine_service`、`bulk_service` 等需要拼装请求的处理器。

## 常见用途

- 子服务需要复用主服务参数（如 `session_user_id`、筛选条件）时开启。
- 避免在 `service` 节点里重复手工映射大量字段。

## 执行阶段（低代码视角）

- 参数/结果处理阶段：在发起“子服务调用”前生效，决定子服务入参拼装方式。

## 怎么用

### 配置位置

- `handler_params[].append_param`
- `result_handler[].params.append_param`
- `batch.append_param`

## 示例

```yml
handler_params:
  - key: service2field
    service:
      service: system.get_page_data_dict
    append_param: true
    save_field: frontend_doc_group_code
```

来源文件：`/data/project/sport/collect/system/schema_page_data/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`append_param`
