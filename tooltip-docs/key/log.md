# log

## 作用

- `log: true` 表示开启该服务运行日志记录。
- `log: false` 或不配置，表示关闭/减少该服务日志输出。
- 你当前规则里：`log` 就是“是否打日志”开关。

## 常见用途

- `log: true` 表示开启该服务运行日志记录。
- `log: false` 或不配置，表示关闭/减少该服务日志输出。

## 执行阶段（低代码视角）

- HTTP 入口与治理阶段：在进入业务执行链前决定访问与日志策略。

## 怎么用

### 配置位置
- 服务节点根级（`service[].log`）

## 示例

```yml
- key: qing_doc_price_save_all_data
  http: true
  log: true
  module: empty
  handler_params:
    - key: group_by
      foreach: "[doc_list]"
```

```yml
- key: qing_doc_price_query
  http: true
  log: false
  module: sql
```

## 注意事项

- 开启日志便于排查参数流转、模板渲染、SQL执行问题。
- 生产环境请避免记录敏感信息（token、密码、隐私字段）。
- 大流量服务建议按需开启，避免日志量过大影响定位效率。

## 元信息

- 来源：`服务定义通用字段`
- 页面标题：`log(服务日志开关)`
