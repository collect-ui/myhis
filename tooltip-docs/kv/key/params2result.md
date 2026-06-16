# key=params2result

## 作用

- 将参数中的多个字段返回

## 常见用途

- 将参数中的多个字段返回

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | params2result |
| fields | array | 是 | 返回参数的内容 |
| fields[from] | template | 是 | 来源哪个字段，支持[你的变量] |
| fields[to] | string | 是 | 目标字段 |

## 示例

```yml
handler_params:
  - key: params2result
    fields:
      - from: "{{.userid}}"
        to: userid
      - from: "{{.user_id}}"
        to: user_id
      - from: "{{.nick}}"
        to: nick
      - from: "{{.username}}"
        to: username
```

## 注意事项

- 将参数中的多个字段返回

## 元信息

- 来源：`服务文档 -> 参数处理 -> 8.params2result / 多参数转结果`
- 页面标题：`params2result(多参数转结果)`
