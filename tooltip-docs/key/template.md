# template

## 作用

- `template` 用于按上下文参数动态生成值。
- 常见位置：
  - `params[字段].template`
  - `params[字段].check.template`
  - `handler_params[].template`
  - `result_handler[].params.template`
- 模板语法基于 Go `text/template`，并扩展了项目自定义函数（如 `must`、`current_date_time`、`get_key` 等）。

## 常见用途

- `template` 用于按上下文参数动态生成值。
- 常见位置：

## 执行阶段（低代码视角）

- 参数/处理器执行阶段：在参数计算、条件判断或处理器运行时生效。

## 怎么用

### 配置位置
- 请按字段所在节点配置。

## 示例

```yml
params:
  search:
    template: "{{ if .search }}%{{.search}}%{{ end }}"
  create_time:
    template: "{{current_date_time}}"

handler_params:
  - key: service2field
    enable: "{{must .price_doc_qing_tree_id}}"
    service:
      service: autodesk.qing_children_ids
      price_doc_qing_tree_id: "[price_doc_qing_tree_id]"
```

## 注意事项

- `template` 生成值后，会写回当前字段或当前处理器目标字段。
- 建议模板尽量纯函数化，不依赖隐式副作用，便于排障。
- 复杂表达式建议拆成中间字段（`save_field`）后再复用。

## 元信息

- 来源：`服务文档 -> 参数/处理器通用字段`
- 页面标题：`template(模板表达式)`
