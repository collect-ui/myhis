# module=model_update

## 作用

- 示例中是个假删除，将一批记录is_delete 改为1
- 主要解决针对数据表的修改，可以单个修改和批量统一修改，主要看你条件怎么写
- 注意* 不能，批量不同记录不同值修改，如果需要可以用model_upsert 批量记录不同值修改
- 支持批量和单行记录修改，主要取决你过滤条件

## 常见用途

- 示例中是个假删除，将一批记录is_delete 改为1
- 主要解决针对数据表的修改，可以单个修改和批量统一修改，主要看你条件怎么写

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| table | string | 是 | 数据库表名 |
| filter | json | 是 | fitler.你的字段__操作符，如果没有__表示进准匹配。操作符号支持__in、__isnull。filter 主要作用就是定位数据。必须有个filter 误配置进行全表更新 |
| filter.你的field__in | string |  | key表示 你的字段 进行in 操作，值表示 ：[你的参数变量]，取参数中的哪个值 |
| filter.你的field__isnull | string |  | key表示 你的字段 进行判读是否为空的操作。主要进行全表修改，比如创建时间为空全部修改 |
| update_fields | array |  | 只更新哪些字段 |
| options | string |  | 支持update_fields动态取变量，[你参数变量] |
| ignore_fields | array |  | 忽略哪些字段，比如用户修改表单里面传了密码，但是要求密码不能改 |

## 示例

```yml
- key: doc_delete
  http: true
  module: model_update
  params:
    collect_doc_id_list:
      check:
        template: "{{must .collect_doc_id_list}}"
        err_msg: 文档不能为空
    is_delete:
      default: "1"
  table: collect_doc
  filter:
    collect_doc_id__in: "[collect_doc_id_list]"
```

## 注意事项

- 支持批量和单行记录修改，主要取决你过滤条件
- 支持控制只改部分字段

## 元信息

- 来源：`服务文档 -> 模块处理 -> 4.model_update / 表修改行数据`
- 页面标题：`model_update(表修改行数据)`
