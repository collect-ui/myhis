# result_handler

## 作用

- `result_handler` 是“模块执行后”的结果处理链。
- 主要用于对模块返回结果做格式化、补充字段、导出、异步通知等后处理。
- 常见场景：
  - `param2result` / `add_param` 重新组织输出
  - `result2excel` + `file_response` 导出下载
  - `hook` 异步触发后续服务（不阻塞主流程）

## 常见用途

- `result_handler` 是“模块执行后”的结果处理链。
- 主要用于对模块返回结果做格式化、补充字段、导出、异步通知等后处理。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | 运行哪个参数处理器，核心字段，根据这个参数决定运行哪个代码 |
| enable | template/string | 否 | 是否执行当前处理器；不配默认执行 |
| template | template | 否 | 对处理结果做成功性判断；返回 false 则失败 |
| err_msg | template/string | 否 | `template` 判断失败时的错误提示 |
| save_field | string | 否 | 把处理结果保存到参数上下文字段（按处理器能力） |

## 示例

```yml
handler_params:
  - key: service2field
    service:
      service: project.service_query
    save_field: project_list
result_handler:
  - key: add_param
    params:
      from_field: project_list
      to_field: project_list

  - key: param2result
    field: project_list

  - key: hook
    enable: "{% if project_list %}True{% endif %}"
    params:
      result_name: _
      service:
        service: message.reinitalertcache
```

## 注意事项

- 执行时机在模块之后，适合做“对外返回形态”与“旁路动作”处理。
- 与 `handler_params` 一样，`result_handler` 是数组，可接入多个处理器按顺序执行。
- `result_handler` 中同样可以调用内部服务（如 `hook`、`combine_service`），实现结果增强与流程联动。

## 元信息

- 来源：`服务文档 -> 如何编写服务 -> 3.什么是参数处理 / 服务描述`
- 页面标题：`什么是参数处理(服务描述)`
