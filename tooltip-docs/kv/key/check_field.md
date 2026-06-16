# key=check_field

## 作用

- 检查字段是否合法，params 中的check只能在一开始检查已有的字段，这个可以支持handler_params中运行其他服务再检查
- 我们经常遇到xx字段不能为空
- 检查服务字段是否合法

## 常见用途

- 检查字段是否合法，params 中的check只能在一开始检查已有的字段，这个可以支持handler_params中运行其他服务再检查
- 我们经常遇到xx字段不能为空

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | check_field 检查params中的字段 |
| fields | array | 是 | 校验字段内容 |
| fields[field] | string | 是 | 字段名称 |
| fields[template] | template | 是 | 校验模板，true表示正常 |
| fields[err_msg] | template | 是 | 失败之后的错误提示 |

## 示例

```yml
handler_params:
  - key: check_field
    fields:
      - field: doc_collect_id
        template: "{{must .doc.collect_doc_id}}"
        err_msg: "文档ID不能为空"
      - field: type
        template: "{{must .doc.type}}"
        err_msg: "文档类型不能为空"
      - field: parent_dir
        template: "{{must .doc.parent_dir}}"
        err_msg: "文档上级目录不能为空"
```

## 注意事项

- 检查服务字段是否合法

## 元信息

- 来源：`服务文档 -> 参数处理 -> 4.check_field / 检查普通字段`
- 页面标题：`check_field(检查普通字段)`
