# key=update_array

## 作用

- 更新参数中数组里面对象字段
- 一般用于批量保存，批量修改，批量服务流程化中
- 比如批量新增生成唯一的uuid，生成创建人，创建时间
- 批量更新数组

## 常见用途

- 更新参数中数组里面对象字段
- 一般用于批量保存，批量修改，批量服务流程化中

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | update_array |
| foreach | string | 是 | 循环数组的对象，[你数组变量] |
| item | string | 是 | 循环时候模板变量名 |
| fields | array | 是 | 字段内容 |
| fields[field] | string | 是 | 字段名 |
| fields[fields[field]] | template | 是 | 字段渲染的模板内容，item是当前行数据，可以直接取params中的变量 |

## 示例

```yml
handler_params:
  - key: update_array
    foreach: "[ldap_group_list]"
    item: item
    fields:
      - field: ldap_group_id
        template: "{{ uuid }}"
      - field: name
        template: "{{.item.after}}"
      - field: has_group
        template: "1"
      - field: last_sync_time
        template: "{{current_date_time}}"
```

## 注意事项

- 批量更新数组

## 元信息

- 来源：`服务文档 -> 参数处理 -> 5.update_array / 更新数组`
- 页面标题：`update_array(更新数组)`
