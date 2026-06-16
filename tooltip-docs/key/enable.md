# enable

## 作用

- `enable` 用于控制“当前处理器是否执行”。
- 典型场景：
  - 条件触发某个处理器（如只有传了某字段才校验/补参）
  - 避免不必要的服务调用（例如 `service2field` 的条件查询）
  - 在同一条服务链里分支化处理逻辑

## 常见用途

- `enable` 用于控制“当前处理器是否执行”。
- 典型场景：

## 执行阶段（低代码视角）

- 参数/处理器执行阶段：在参数计算、条件判断或处理器运行时生效。

## 怎么用

### 配置位置
- `handler_params` 每个处理器项内
- `result_handler` 每个处理器项内
- 部分子规则项（如 `fields` 中）也支持 `enable/ifTemplate` 条件控制

### 取值规则
- 不配置 `enable`：默认执行该处理器。
- 配置 `enable`：
  - 结果为真（`True/true/1`）时执行
  - 结果为假（`False/false/0/空`）时跳过
- 推荐：
  - 频繁判断可先算出布尔变量，再写 `enable: "[enable]"`，可读性更高。
  - 复杂条件再用模板表达式。

## 示例

```yml
handler_params:
  - key: service2field
    enable: "{% if project_code %}True{% endif %}"
    service:
      service: project.project_query
      project_code: "[project_code]"
    save_field: project_info

result_handler:
  - key: hook
    enable: "{% if project_info %}True{% endif %}"
    params:
      service:
        service: message.reinitalertcache
```

## 注意事项

- `enable` 仅决定“是否执行该项”，不负责结果正确性校验；结果校验请使用 `template + err_msg`。
- 同一服务里多个处理器按顺序执行，前一项写入的参数可以被后一项 `enable` 使用。

## 元信息

- 来源：`服务编写 -> 参数处理/结果处理通用字段`
- 页面标题：`enable(条件执行开关)`
