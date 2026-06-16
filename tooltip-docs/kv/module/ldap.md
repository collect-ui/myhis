# module=ldap

## 作用

- 支持ldap模块的增删改查对对对，示例中config_detail_query 是查询ldap 的配置，我将ldap配置到数据库了
- 本模块不太属于公共模块，只是我工作中需要对接，属于业务模块没有过多介绍，至于参数配置请到https://github.com/go-ldap/ldap
- 支持ldap增删改查
- 支持require 引入公共文件

## 常见用途

- 支持ldap模块的增删改查对对对，示例中config_detail_query 是查询ldap 的配置，我将ldap配置到数据库了
- 本模块不太属于公共模块，只是我工作中需要对接，属于业务模块没有过多介绍，至于参数配置请到https://github.com/go-ldap/ldap

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | ldap,运行ldap模块 |
| data_file | string | 是 | ldap 配置路径地址 |
| data_file.connection | json | 是 | ldap 连接信息 |
| data_file.connection.server | string | 是 | 服务器地址 |
| data_file.connection.user | string | 是 | 登陆的用户信息 |
| data_file.connection.password | string | 是 | 登陆的密码 |
| data_file.method | string | 是 | 执行方法。search:搜索，add:添加；modify:修改;delete:删除;modifyDn修改dn信息 |
| data_file.SearchParams | json |  | 搜索对应的参数，具体参数看示例 |
| data_file.AddParams | json |  | 添加对应的参数，具体参数看示例 |
| data_file.ModifyParams | json |  | 修改对应的参数，具体参数看示例 |
| data_file.DeleteParams | json |  | 删除对应的参数，具体参数看示例 |
| data_file.ModifyDnParams | json |  | 修改ou对应的参数 |

## 示例

```yml
- key: ldap_search
  http: true
  module: ldap
  params:
    search_username:
      check:
        template: "{{must .search_username}}"
        err_msg: 用户名不能为空
  handler_params:
    - key: service2field
      service:
        service: config.conf单独ig_detail_query
        group_name: ldap_config
      save_field: ldap_config
      template: "{{must .ldap_config}}"
      err_msg: ldap配置不存在
  data_file: search.json
```

## 注意事项

- 支持ldap增删改查
- 支持require 引入公共文件

## 元信息

- 来源：`服务文档 -> 模块处理 -> 10.ldap / ldap`
- 页面标题：`ldap(ldap)`
