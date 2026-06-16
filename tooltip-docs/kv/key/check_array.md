# key=check_array

## 作用

- 对目标数组逐项执行校验模板，命中失败即返回错误。

## 常见用途

- 对目标数组逐项执行校验模板，命中失败即返回错误。

## 执行阶段（低代码视角）

- 参数处理阶段：在模块执行前运行，用于加工/校验/补充参数。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| foreach | string/template | 是 | 要校验的数组字段 |
| item | string | 是 | 当前项变量名 |
| fields | array | 是 | 校验规则列表 |
| fields[].template | string | 是 | 校验模板，返回 false 则失败 |
| fields[].err_msg | string | 是 | 失败提示模板 |

## 示例

```yml
handler_params:
  - key: check_array
    foreach: detail_list
    item: detail
    fields:
      - template: "{% if detail.name %}True{% endif %}"
        err_msg: "第{{item_order_index}}行名称不能为空"
```

## 注意事项

- 框架会注入 `item_order_index`（从 1 开始），便于错误提示定位。
- 任一规则失败会立即返回，不继续校验后续项。

### 源码定位
- Python类路径：`collect.service_imp.request_handlers.handlers.check_array.CheckArray`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/request_handlers/handlers/check_array.py`

## 元信息

- 来源：`handler_params -> check_array / 数组逐项校验`
- 页面标题：`check_array(数组逐项校验)`
