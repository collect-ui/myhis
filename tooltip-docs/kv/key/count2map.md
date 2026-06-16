# key=count2map

## 作用

- 一般是sql查询的count转字段
- 这个用比较少，基本用不着count

## 常见用途

- 一般是sql查询的count转字段
- 这个用比较少，基本用不着count

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | count2map |
| field | template | 是 | 结果里面的字段 |

## 示例

```yml
handler_params:
  - key: count2map
```

## 注意事项

- 一般是sql查询的count转字段
- 这个用比较少，基本用不着count

## 元信息

- 来源：`服务文档 -> 参数处理 -> 18.count2map / count 转字段`
- 页面标题：`count2map(count 转字段)`
