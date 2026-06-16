# key=param2result

## 作用

- 参数中字段转结果，一般用于只需要返回一个字段
- 参数中的某个字段转结果

## 常见用途

- 参数中字段转结果，一般用于只需要返回一个字段
- 参数中的某个字段转结果

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | param2result |
| field | string | 是 | [你参数变量] |

## 示例

```yml
result_handler:
  - key: param2result
    field: "[access_token]"
```

## 注意事项

- 参数中的某个字段转结果

## 元信息

- 来源：`服务文档 -> 参数处理 -> 7.param2result / 参数转结果`
- 页面标题：`param2result(参数转结果)`
