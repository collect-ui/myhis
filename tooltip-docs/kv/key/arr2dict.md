# key=arr2dict

## 作用

- 数据列表里面，套了一个二级数组，将二级数组转对象
- 是转key /value 对象
- 比如参数管理

## 常见用途

- 数据列表里面，套了一个二级数组，将二级数组转对象
- 是转key /value 对象

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string |  | arr2dict |
| foreach | string | 是 | 循环的数组 |
| children | string |  | 如果有children 表示有个二级数组，转二级字段里面对象 |
| result_name | string |  | 转换的结果对象，二级数组有效 |
| field | string |  | 字段名[你的变量] |
| value | string |  | 字段值[你的变量] |

## 示例

```yml
- key: arr2dict
  name: 如果有children 表示有个二级数组
  enable: "{{must .config}}"
  foreach: "[config]"
  children: "children"
  result_name: "children_config"
  field: "[name]"
  value: "[value]"
  save_field: config
```

## 注意事项

- 数据列表里面，套了一个二级数组，将二级数组转对象
- 是转key /value 对象
- 比如参数管理

## 元信息

- 来源：`服务文档 -> 参数处理 -> 24.arr2dict / 数组转对象`
- 页面标题：`arr2dict(数组转对象)`
