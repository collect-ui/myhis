# handler_params

## 作用

- `handler_params` 是“模块执行前”的请求参数处理链。
- 主要用于对前台传入数据做二次加工，使其满足后续模块执行条件。
- 典型场景：
  - 编码唯一性校验（先查库再判断）
  - 把 ID 数组转换成对象数组，供批量入库
  - 根据字段标志拆分数据，分别调用多个内部服务保存
- `handler_params` 和 `result_handler` 都是数组，可串联多个处理器，并可通过 `service2field` 调内部任意服务。

## 常见用途

- `handler_params` 是“模块执行前”的请求参数处理链。
- 主要用于对前台传入数据做二次加工，使其满足后续模块执行条件。

## 执行阶段（低代码视角）

- 参数处理阶段：在主体模块执行前运行。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | 运行哪个参数处理器，核心字段，根据这个参数决定运行哪个代码 |
| enable | template/string | 否 | 是否执行当前处理器；不配默认执行 |
| template | template | 否 | 对处理器结果做成功性判断；返回 false 则失败 |
| err_msg | template/string | 否 | `template` 判断失败时的错误提示 |
| save_field | string | 否 | 处理器输出保存到哪个参数字段 |

## 示例

```yml
handler_params:
  - key: service2field
    enable: "{% if project_code %}True{% endif %}"
    service:
      service: project.service_query
      project_code: "[project_code]"
    save_field: project_list

  - key: check_field
    template: "{% if project_list %}True{% endif %}"
    err_msg: 项目不存在

result_handler:
  - key: param2result
    field: project_list
```

## 注意事项

- 执行顺序：`handler_params` -> `module` -> `result_handler`。
- `handler_params` 的每一项可依赖上一项写入的参数（通过 `save_field` 共享上下文）。
- 大型编辑场景（一次提交需要落 20~30 张表）通常依赖 `handler_params` 先拆分、补齐、路由数据，再由多个内部服务落库。

## 元信息

- 来源：`服务文档 -> 如何编写服务 -> 3.什么是参数处理 / 服务描述`
- 页面标题：`什么是参数处理(服务描述)`
