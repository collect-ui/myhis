# key=add_param

## 作用

- 从当前参数上下文读取一个值，写入当前结果对象（或结果数组中的每个对象）。
- 常用于把请求参数、上游 handler 生成的字段回填到输出结果。

## 常见用途

- 从当前参数上下文读取一个值，写入当前结果对象（或结果数组中的每个对象）。
- 常用于把请求参数、上游 handler 生成的字段回填到输出结果。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.from_field | string/template | 是 | 来源字段或模板表达式 |
| params.to_field | string | 是 | 目标结果字段名 |

## 示例

```yml
result_handler:
  - key: add_param
    params:
      from_field: "{{.project_code}}"
      to_field: project_code
```

## 注意事项

- 对象结果：直接写入 `result[to_field]`。
- 数组结果：逐项写入 `item[to_field]`。
- `from_field` 使用模板渲染，可取复杂路径。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.add_param.AddParam`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/add_param.py`

## 元信息

- 来源：`result_handler -> add_param / 把参数值写入结果`
- 页面标题：`add_param(把参数值写入结果)`
