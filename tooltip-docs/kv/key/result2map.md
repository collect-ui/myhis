# key=result2map

## 作用

- 无
- 就是将结果，外层在包一层对象
- 这个用比较少，一般可能利用params2result

## 常见用途

- 无
- 就是将结果，外层在包一层对象

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | params2result |
| field | template | 是 | 结果字段的key |

## 示例

```yml
handler_params:
  - key: result2map
```

## 注意事项

- 无
- 就是将结果，外层在包一层对象
- 这个用比较少，一般可能利用params2result

## 元信息

- 来源：`服务文档 -> 参数处理 -> 17.result2map / 结果转字段`
- 页面标题：`result2map(结果转字段)`
