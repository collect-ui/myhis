# module=service_flow

## 作用

- 像工作流一样的运行多个服务，运行一个节点服务，通过计算，运行下一个服务节点
- 服务流程化思想来源于工作流
- 接触到loonflow工作流，我一直试想着，我写的代码能不能像一个工作流一样流转，我们只写一小部分节点，通过工作流流转，运行到下个服务
- 比如我写个新建用户，然后流转到新建角色，如果中途失败，流转到删除用户、删除角色然后返回

## 常见用途

- 像工作流一样的运行多个服务，运行一个节点服务，通过计算，运行下一个服务节点
- 服务流程化思想来源于工作流

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | service_flow ,服务流程化 |
| data_json | string | 是 | 服务配置地址 |
| data_json.finish | json |  | 和handler_params 通样配置，无论流程成功与失败都会运行 |
| data_json.services | array | 是 | 服务流程的节点， |
| data_json.services[node_key] | string | 是 | 流程的关键字，流程必须包含start,end |
| data_json.services[node_type] | string | 是 | start 开始，node节点，end结束 |
| data_json.services[name] | string | 是 | 节点名称 |
| data_json.services[key] | string | 是 | 剩下的字段和handler_params一致，字段取决与具体【参数处理】，开始和结束不需要中间都要key |
| data_json.services[node_next] | template | 是 | 运行完成后，下个节点的状态 |
| data_json.services[ignore_error] | boolean |  | 是否忽略错误 |
| data_json.services[node_fail] | template |  | 失败后运行的下个状态 |

## 示例

```yml
- key: system_login
  module: service_flow
  params:
    username:
      check:
        template: "{{must .username}}"
        err_msg: 用户名不能为空
    password_md5:
      check:
        template: "{{must .password}}"
        err_msg: 密码不能为空
      template: "{{md5 .password}}"
    has_user:
      default: true
  data_json: system_login.json
  result_handler:
    - key: param2result
      field: "[session_user]"
```

## 注意事项

- 服务流程化

## 元信息

- 来源：`服务文档 -> 模块处理 -> 11.service_flow / 服务流程化`
- 页面标题：`service_flow(服务流程化)`
