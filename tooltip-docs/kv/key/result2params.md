# key=result2params

## 作用

- 将运行的结果转参数
- 执行完成结果后，结果还需要进行处理，比如去重，结合拼接一些字段
- 前面是param2result 参数转结构，同样也支持结果转参数
- 结果转参数

## 常见用途

- 将运行的结果转参数
- 执行完成结果后，结果还需要进行处理，比如去重，结合拼接一些字段

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | result2params |
| fields | array | 是 | 需要转参数的字段内容 |
| fields[to] | string | 是 | 转到参数的哪个字段,[你的变量] |
| fields[from] | string |  | 如果没有配置值from,将整个结果转字段，如果配置from，这将结果里面字段转参数。[你的结果变量] |

## 示例

```yml
result_handler:
  - key: result2params
    fields:
      - to: "[config_params]"
```

## 注意事项

- 结果转参数

## 元信息

- 来源：`服务文档 -> 参数处理 -> 16.result2params / 结果转参数`
- 页面标题：`result2params(结果转参数)`
