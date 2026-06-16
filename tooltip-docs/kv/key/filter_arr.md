# key=filter_arr

## 作用

- 过滤数组，一般把增、删、改的数据过滤，分别对应操作
- 从数组中过滤数组

## 常见用途

- 过滤数组，一般把增、删、改的数据过滤，分别对应操作
- 从数组中过滤数组

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | filter_arr |
| foreach | string | 是 | 循环数组对象取值，[你的变量] |
| item | string | 是 | item 变量取值 |
| if_template | template | 是 | 过滤条件，true 保留下来 |

## 示例

```yml
- key: filter_arr
  foreach: "[change_list]"
  item: item
  if_template: "{{and (eq .item.operation \"remove\") (ne .item.has_group \"0\") }}"
  save_field: remove_list
```

## 注意事项

- 从数组中过滤数组

## 元信息

- 来源：`服务文档 -> 参数处理 -> 22.filter_arr / 过滤数组`
- 页面标题：`filter_arr(过滤数组)`
