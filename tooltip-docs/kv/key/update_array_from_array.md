# key=update_array_from_array

## 作用

- 从另外一个数组，找到相同的记录，本更新字段
- a数组从b数组里面更新字段,治取交集部分更新
- 数组更新数组

## 常见用途

- 从另外一个数组，找到相同的记录，本更新字段
- a数组从b数组里面更新字段,治取交集部分更新

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | update_array_from_array |
| foreach | array | 是 | 左边循环的数组，待更新的数组 |
| item | string | 是 | 左边对象名称取值 |
| field | string | 是 | 左边匹配关键字段[你的item变量] 支持& 取2个字段 |
| right | string | 是 | 右边数组 |
| right_field | string | 是 | 右边匹配关键字段[你的item变量] 支持& 取2个字段 |
| fields | array | 是 | 更新字段的内容 |
| fields[field] | template | 是 | 字段名称 |
| fields[template] | template | 是 | 支持[你的变量]，左边的用item,右边的用right |

## 示例

```yml
- key: update_array_from_array
  name: "根据group+name 更新操作，有修改，没有就新增"
  foreach: "[change_list]"
  item: item
  field: "[group_id&name_copy]"
  right: "[local_detail_list]"
  right_field: "[group_id&name]"
  fields:
    - field: operation
      name: 如果操作是删除，还是原来的删除，如果是新增操作，存在这改为修改
      template: "{{ if eq .item.operation \"add\"}}modify{{else}}{{.item.operation}}{{end}}"
    - field: config_detail_id
      template: "[right.config_detail_id]"
```

## 注意事项

- 数组更新数组

## 元信息

- 来源：`服务文档 -> 参数处理 -> 28.update_array_from_array / 更新数组`
- 页面标题：`update_array_from_array(更新数组)`
