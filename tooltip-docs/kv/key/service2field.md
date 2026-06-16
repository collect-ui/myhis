# key=service2field

## 作用

- 将另外一个服务执行结果作为本服务的一个参数
- 具体服务运行的什么没有任何限制，可以是增删除改查
- 比如校验这个编码是否唯一，我们需要查询数据库表来判断，对比行记录，需要将之前的记录查询出来 处理有参数处理的公共字段，还有额外字段，service，append_param,a...
- 可以调用其他服务，作为本服务的入参

## 常见用途

- 将另外一个服务执行结果作为本服务的一个参数
- 具体服务运行的什么没有任何限制，可以是增删除改查

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| service | json | 是 | json下面service 调用目标服务，如何需要引入参数变量,[你的变量] |
| append_param | boolean |  | 是否拼接全部参数,默认不拼接 |
| append_item_param | boolean |  | 默认拼接某个字段的参数 |
| item | string |  | 如果拼接某个字段的参数，字段的取值，[你的变量] |

## 示例

```yml
handler_params:
  - key: service2field
    service:
      service: hrm.ldap_add
    append_param: true
```

## 注意事项

- 可以调用其他服务，作为本服务的入参
- 可以结合template 来校验，校验服务是否正常

## 元信息

- 来源：`服务文档 -> 参数处理 -> 2.service2field / 服务转字段`
- 页面标题：`service2field(服务转字段)`
