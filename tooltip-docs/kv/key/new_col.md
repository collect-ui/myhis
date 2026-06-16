# key=new_col

## 作用

- 给结果对象或对象数组按模板新增字段。
- 支持对子节点（如 children）批量新增字段，也支持新增后删除临时字段。

## 常见用途

- 给结果对象或对象数组按模板新增字段。
- 支持对子节点（如 children）批量新增字段，也支持新增后删除临时字段。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.to_field | array | 是 | 新增字段规则列表 |
| params.to_field[].field | string | 是 | 新增字段名 |
| params.to_field[].template | string | 否 | 字段值模板 |
| params.field | string | 否 | 子数组字段名（如 children），配了后在子节点上处理 |
| params.remove | array | 否 | 处理完成后删除的字段列表 |

## 示例

```yml
result_handler:
  - key: new_col
    params:
      to_field:
        - field: display_name
          template: "{{.name}}({{.id}})"
```

## 注意事项

- 模板上下文会合并“全局参数 + 当前对象字段”。
- 子节点处理时会临时注入 `parent` 便于模板访问父节点。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.new_col.NewCol`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/new_col.py`

## 元信息

- 来源：`result_handler -> new_col / 结果新增列`
- 页面标题：`new_col(结果新增列)`
