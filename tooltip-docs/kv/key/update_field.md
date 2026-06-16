# key=update_field

## 作用

- 更新params中的字段
- 更新参数中的字段

## 常见用途

- 更新params中的字段
- 更新参数中的字段

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | update_field |
| fields | array | 是 | 更新的字段内容 |
| fields[field] | string | 是 | 字段名称 |
| fields[template] | template | 是 | 字段内容,支持模板生成变量，支持[]取参数中值 |

## 示例

```yml
handler_params:
  - key: update_field
    name: 更新字段
    fields:
      - field: user_info
        template: "[modify_data.user_info]"
      - field: change_list
        template: "[modify_data.change_list]"
```

## 注意事项

- 更新参数中的字段

## 元信息

- 来源：`服务文档 -> 参数处理 -> 3.update_field / 更新普通字段`
- 页面标题：`update_field(更新普通字段)`
