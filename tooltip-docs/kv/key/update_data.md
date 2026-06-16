# key=update_data

## 作用

- 遍历一个目标数组（`foreach`），按 `fields` 模板批量更新每个元素字段。
- 常用于“先查出列表，再按模板补充/重算字段”。

## 常见用途

- 遍历一个目标数组（`foreach`），按 `fields` 模板批量更新每个元素字段。
- 常用于“先查出列表，再按模板补充/重算字段”。

## 执行阶段（低代码视角）

- 参数处理阶段：在模块执行前运行，用于加工/校验/补充参数。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| foreach | string | 是 | 目标数组字段名（从 params 读取） |
| item | string | 否 | 当前循环项变量名；不配时直接用当前项作为模板上下文 |
| fields | array | 是 | 字段更新规则列表 |
| fields[].field | string | 是 | 要写入的字段名 |
| fields[].template | string | 是 | 渲染模板 |
| fields[].enable | string/template | 否 | 为 false 时跳过该字段更新 |

## 示例

```yml
handler_params:
  - key: update_data
    foreach: issue_list
    item: issue
    fields:
      - field: owner_name
        template: "{{.issue.owner.display_name}}"
```

## 注意事项

- 会注入 `item_order_index`（从 1 开始）供模板使用。
- 目标数组元素必须是对象（dict），否则报错。
- 任一 `fields` 规则缺少 `template` 会直接失败。

### 源码定位
- Python类路径：`collect.service_imp.request_handlers.handlers.update_data.UpdateData`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/request_handlers/handlers/update_data.py`

## 元信息

- 来源：`handler_params -> update_data / 批量更新数组对象字段`
- 页面标题：`update_data(批量更新数组对象字段)`
