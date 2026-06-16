# module=model_delete

## 作用

- 针对表数据库删除

## 常见用途

- 针对表数据库删除

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | module: model_delete |
| table | string | 是 | 数据库的表名 |
| filter | string | 是 | 参考model_update 的fitler |

## 示例

```yml
- key: role_ldap_group_delete
  module: model_delete
  params:
    role_id_list:
      check:
        template: "{{must .role_id_list}}"
        err_msg: 角色不能为空
  table: "role_ldap_group"
  filter:
    role_id__in: "[role_id_list]"
```

## 注意事项

- 针对表数据库删除

## 元信息

- 来源：`服务文档 -> 模块处理 -> 5.model_delete / 表数据删除`
- 页面标题：`model_delete(表数据删除)`
