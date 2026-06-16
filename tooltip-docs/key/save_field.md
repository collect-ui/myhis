# save_field

## 作用

- 把当前处理器产出的结果，保存到“参数上下文”里的某个字段。
- 这个字段可在后续处理器、模块执行、结果处理中继续引用。

## 常见用途

- 调内部服务取补充数据，保存为中间变量（如 `user_info`、`project_list`）。
- 先做转换/过滤，再把结果供后续处理器继续消费。
- 把模板渲染结果保存为消息体、文件路径、导出结果等。

## 执行阶段（低代码视角）

- 参数/处理器执行阶段：在参数计算、条件判断或处理器运行时生效。

## 怎么用

### 配置位置
- `handler_params` 的处理器项中（最常见）
- `result_handler` 的处理器项中（按处理器能力）
- 常见于 `service2field`、`filter_arr`、`render_template`、`combine_service` 等

## 示例

```yml
handler_params:
  - key: service2field
    service:
      service: jira.issue_commit_count
      project_code: "[project_code]"
    save_field: issue_commit_list

result_handler:
  - key: add_param
    params:
      from_field: issue_commit_list
      to_field: issue_commit_list
```

## 注意事项

- `save_field` 命名建议明确业务语义，避免覆盖原始入参（除非你明确要覆盖）。
- 若未配置 `save_field`，处理器行为取决于实现：
  - 有的处理器直接改原对象
  - 有的处理器仅返回结果但不持久落参
  - 有的处理器要求必须配置 `save_field`

## 元信息

- 来源：`服务编写 -> 参数处理/结果处理通用字段`
- 页面标题：`save_field(结果落参字段)`
