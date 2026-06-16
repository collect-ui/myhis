# key=update_data_from_array

## 作用

- 用来源数组（`from_list`）匹配目标数组（`foreach`），并按字段模板批量回填。
- 支持两种匹配方式：模板匹配、键字段匹配（推荐）。

## 常见用途

- 用来源数组（`from_list`）匹配目标数组（`foreach`），并按字段模板批量回填。
- 支持两种匹配方式：模板匹配、键字段匹配（推荐）。

## 执行阶段（低代码视角）

- 参数处理阶段：在模块执行前运行，用于加工/校验/补充参数。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.foreach | string | 是 | 目标数组字段名 |
| params.item | string | 是 | 目标数组当前项变量名 |
| params.from_list | string | 是 | 来源数组字段名，支持 `a.b` 形式 |
| params.from_item | string | 是 | 来源数组当前项变量名 |
| params.fields | array | 是 | 回填字段规则 |
| params.fields[].field | string | 是 | 回填字段名 |
| params.fields[].template | string | 是 | 回填模板 |
| params.ifTemplate | string | 否 | 模板匹配规则（方式1） |
| params.from_list_fields | array | 否 | 来源数组匹配键字段列表（方式2） |
| params.foreach_key_fields | array | 否 | 目标数组匹配键字段列表（方式2） |
| params.second_item | string | 条件必填 | `from_list` 为二级数组时使用 |

## 示例

```yml
handler_params:
  - key: update_data_from_array
    params:
      foreach: issue_list
      item: issue
      from_list: user_list
      from_item: user
      from_list_fields: [user_id]
      foreach_key_fields: [assignee_id]
      fields:
        - field: assignee_name
          template: "{{.user.name}}"
```

## 注意事项

- 必须至少配置一种匹配规则：`ifTemplate` 或 `from_list_fields + foreach_key_fields`。
- 键匹配方式会先构建来源字典，性能高于双层循环模板匹配。

### 源码定位
- Python类路径：`collect.service_imp.request_handlers.handlers.update_data_from_array.UpdateDataFromArray`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/request_handlers/handlers/update_data_from_array.py`

## 元信息

- 来源：`handler_params -> update_data_from_array / 按来源数组回填目标数组`
- 页面标题：`update_data_from_array(按来源数组回填目标数组)`
