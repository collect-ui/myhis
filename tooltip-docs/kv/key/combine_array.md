# key=combine_array

## 作用

- 一般利用2个服务结合，将一个数字拼接到另外一个数组中取，比如参数管理，二级数组
- 将数组拼接到另外一个数组中去，将右边的数据拼接到左边

## 常见用途

- 一般利用2个服务结合，将一个数字拼接到另外一个数组中取，比如参数管理，二级数组
- 将数组拼接到另外一个数组中去，将右边的数据拼接到左边

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | combine_array |
| foreach | string | 是 | 左边的数组 |
| field | string | 是 | 左边关键字段 |
| right | string | 是 | 右边的数组 |
| right_field | string | 是 | 右边的关键字段 |
| children | string | 是 | 生成的字段名称 |

## 示例

```yml
- key: combine_array
  enable: "{{must .width_doc}}"
  foreach: "[group_list]"
  field: "[doc_group_id]"
  right: "[doc_list]"
  right_field: "[parent_dir]"
  children: "children"
```

## 注意事项

- 将数组拼接到另外一个数组中去，将右边的数据拼接到左边

## 元信息

- 来源：`服务文档 -> 参数处理 -> 27.combine_array / 数组结合数组`
- 页面标题：`combine_array(数组结合数组)`
