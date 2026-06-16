# key=val2param

## 作用

- 从当前结果中按模板提取值，并写入请求参数，供后续处理器使用。

## 常见用途

- 从当前结果中按模板提取值，并写入请求参数，供后续处理器使用。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.template | string | 是 | 提取模板（基于每个结果项渲染） |
| params.to_field | string | 是 | 写入 params 的目标字段 |

## 示例

```yml
result_handler:
  - key: val2param
    params:
      template: "{{.issue_key}}"
      to_field: issue_key_list
```

## 注意事项

- 当前结果是对象：写入单值。
- 当前结果是数组：逐项渲染并写入数组。
- 当前结果为空时直接透传，不报错。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.val_2_param.Val2Param`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/val_2_param.py`

## 元信息

- 来源：`result_handler -> val2param / 值转参数`
- 页面标题：`val2param(值转参数)`
