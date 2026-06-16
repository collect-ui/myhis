# http

## 作用

- `http: true` 表示该服务允许通过 HTTP 接口直接访问（对外暴露）。
- `http: false` 或不配置，通常表示仅内部调用（如通过 `service2field/hook/bulk_service`）。
- 你当前规则里：`http` 就是“是否对外 HTTP 接口访问”开关。

## 常见用途

- `http: true` 表示该服务允许通过 HTTP 接口直接访问（对外暴露）。
- `http: false` 或不配置，通常表示仅内部调用（如通过 `service2field/hook/bulk_service`）。

## 执行阶段（低代码视角）

- HTTP 入口与治理阶段：在进入业务执行链前决定访问与日志策略。

## 怎么用

### 配置位置
- 服务节点根级（`service[].http`）

## 示例

```yml
- key: qing_doc_price_query
  http: true
  module: sql
  params:
    page:
      type: int
      default: 1
```

```yml
- key: qing_doc_price_save_all_data
  http: false
  module: empty
  handler_params:
    - key: service2field
      service:
        service: autodesk.qing_doc_price_bulk_create
```

## 注意事项

- 对外服务建议配合登录、token、权限校验配置使用。
- 内部服务即使 `http: false`，仍可被其他服务链调用。
- 若服务需前端直连，务必确认 `http: true` 已设置。

## 元信息

- 来源：`服务定义通用字段`
- 页面标题：`http(是否对外HTTP访问)`
