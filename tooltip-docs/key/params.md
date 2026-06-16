# params

## 作用

- `params` 是服务的参数池（变量区域），本质是一个 `key/value` 的 JSON 对象。
- HTTP 请求入口通常是：`{service: xxx.xxx, ...业务参数}`。
- 请求参数会先进入参数池，再被 `handler_params`（执行前）和 `result_handler`（执行后）持续读写。
- 可以理解为一个“贯穿服务全流程的大箱子”：每个步骤都可往里放数据、从里取数据。
- 典型用途：
  - 简单字段校验（不能为空）
  - 默认值填充（如 `is_delete=0`）
  - 模板生成值（如 `create_time`、`create_user`）
  - 为后续处理器提供中间变量

## 常见用途

- `params` 是服务的参数池（变量区域），本质是一个 `key/value` 的 JSON 对象。
- HTTP 请求入口通常是：`{service: xxx.xxx, ...业务参数}`。

## 执行阶段（低代码视角）

- 参数装载阶段：请求进入服务后先形成参数池，供后续所有步骤读写。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params | json | 是 | 参数定义池，一级 key 就是变量名（如 `user_list`） |
| params[字段] | json | 否 | 单字段规则定义 |
| params[字段].template | template | 否 | 用模板生成字段值 |
| params[字段].exec | bool | 否 | 模板执行模式（仅部分引擎支持） |
| params[字段].default | any | 否 | 默认值，类型不限 |
| params[字段].type | string | 否 | 类型转换（如 `bool/int/float64/string`） |
| params[字段].check | json | 否 | 字段校验规则 |
| params[字段].check.template | template | 是 | 校验表达式，true 通过 / false 失败 |
| params[字段].check.err_msg | template/string | 是 | 校验失败提示信息 |

## 示例

```yml
- key: bulk_create_user
  module: bulk_create
  log: true
  http: true
  params:
    user_list:
      check:
        template: "{{must .user_list}}"
        err_msg: 用户列表不能为空

    is_delete:
      default: 0

    create_time:
      template: "{{current_date_time}}"

    create_user:
      template: "{{.session_user_id}}"

    page:
      type: int
      default: 1
```

## 注意事项

- 请求参数也会直接进入 `params`，并贯穿整个服务流程。
- `handler_params` 与 `result_handler` 都是在读写同一份参数池。
- 默认会注入 `session_user_id`（当前登录用户ID）。

### 执行顺序说明（按引擎区分）
- Python 低代码引擎（collect-py 常见顺序）：
  - `default/type` 预处理 -> `template` 覆盖 -> `check` 校验
  - 实务上通常表现为：`template > 传入值 > default`，最后执行 `check`
- Go 低代码引擎（当前 `sport` 项目）：
  - `default` -> `check` -> `template` -> `type`
  - 以运行版本源码为准，跨项目请注意差异

## 元信息

- 来源：`服务文档 -> 如何编写服务 -> 2.什么是参数 / 服务描述`
- 页面标题：`什么是参数(服务描述)`
