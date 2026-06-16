# key=prop_arr

## 作用

- 过滤对象数组的某个字段转数组，比如传对象数组过来，过滤成简单ID数组，好根据ID删除
- 对象数组转简单数组

## 常见用途

- 过滤对象数组的某个字段转数组，比如传对象数组过来，过滤成简单ID数组，好根据ID删除
- 对象数组转简单数组

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | prop_arr |
| foreach | string | 是 | 数组循环对象，[你的变量] |
| value | string | 是 | value 取值，[你的item的变量] |

## 示例

```yml
handler_params:
  - key: prop_arr
    foreach: "[detail_list]"
    value: "[config_detail_id]"
    save_field: config_detail_id_list
```

## 注意事项

- 对象数组转简单数组

## 元信息

- 来源：`服务文档 -> 参数处理 -> 23.prop_arr / 对象数组转数组`
- 页面标题：`prop_arr(对象数组转数组)`
