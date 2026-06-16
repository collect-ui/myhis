# key=combine_service

## 作用

- 先拿当前服务结果，再调用另一个服务，把返回数据按键关联回当前结果。
- 适合做“关联扩展字段”“批量补充明细”。

## 常见用途

- 先拿当前服务结果，再调用另一个服务，把返回数据按键关联回当前结果。
- 适合做“关联扩展字段”“批量补充明细”。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.service | object | 是 | 要调用的服务节点（含 `service: xxx.xxx`） |
| params.save_field | string | 是 | 回填到当前结果的字段名 |
| params.from_field | string/template | 否 | 被调服务结果里的匹配键 |
| params.to_field | string/template | 否 | 当前结果里的匹配键 |
| params.multiple | bool | 否 | true 时按一对多回填数组 |
| params.append_param | bool | 否 | 调子服务是否拼接当前参数，默认 true |
| params.result_name | string | 否 | 将当前结果注入到参数中的字段名 |
| params.append_to_original | bool | 否 | true 且非 multiple 时，直接把匹配对象字段平铺到当前对象 |

## 示例

```yml
result_handler:
  - key: combine_service
    params:
      service:
        service: jira.user_query
      from_field: "{{.user_id}}"
      to_field: "{{.assignee_id}}"
      save_field: user
      multiple: false
```

## 注意事项

- `from_field/to_field` 支持模板渲染，不是纯静态字段名。
- 没匹配到时：`multiple=false` 回填 `{}`，`multiple=true` 回填 `[]`。
- `from_field` 和 `to_field` 都不配时，会把子服务完整结果写入 `save_field`。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.combine_service.CombineService`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/combine_service.py`

## 元信息

- 来源：`result_handler -> combine_service / 结果结合其他服务`
- 页面标题：`combine_service(结果结合其他服务)`
