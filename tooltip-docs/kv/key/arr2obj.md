# key=arr2obj

## 作用

- 仅仅用于服务的结果是一个数组，将数组转对象，方便其他服务模板取值，不用从数组取，直接.xx.xx 属性
- 数组结果转对象结果

## 常见用途

- 仅仅用于服务的结果是一个数组，将数组转对象，方便其他服务模板取值，不用从数组取，直接.xx.xx 属性
- 数组结果转对象结果

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | arr2obj |
| field | string |  | 参数中的某个字段，不传就是对结果 |

## 示例

```yml
- key: get_user_modify_data
  http: true
  log: true
  module: empty
  modify_config: user_modify.json
  params:
    user_id:
      check:
        template: "{{must .user_id}}"
        err_msg: 用户ID不能为空
    right_ldap_group:
      default: []
  handler_params:
    - key: service2field
      service:
        service: hrm.user_list
        user_id: "[user_id]"
        count: false
        to_obj: true
      save_field: user_info
    - key: service2field
      enable: "{{must .user_info.roles}}"
      service:
        service: hrm.ldap_group_query
        roles: "[user_info.roles]"
      save_field: right_ldap_group
```

## 注意事项

- 数组结果转对象结果

## 元信息

- 来源：`服务文档 -> 参数处理 -> 6.arr2obj / 数组结果转对象`
- 页面标题：`arr2obj(数组结果转对象)`
