# module=model_save

## 作用

- 对数据库表的新增一行记录
- 针对数据库表进行新增一行
- 可以利用params里面的template生成uuid、创建时间、创建人、设置默认时间
- params 下面一级是变量名称，比如示例中create_time 是表里面create_time 字段，下面生成规则，与校验规则

## 常见用途

- 对数据库表的新增一行记录
- 针对数据库表进行新增一行

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | module: model_save 执行新增模块 |
| table | string | 是 | 数据库表名 |
| params | json |  | params 是通用模块，下面key是对应字段 |
| params.你的field | json |  | 请求自定义字段 |
| params.你的field.type | string |  | 转换的数据类型，支持int，bool，int，int32，int64，bigint，float，time.time，time，sql.nulltime |
| params.你的field.template | template |  | 生成数据的模板 |
| params.你的field.check | array |  | 检查数据类型 |
| params.你的field.check.template | template |  | 校验规则模板，true为正常 |
| params.你的field.check.err_msg | template |  | 校验规则失败的消息 |

## 示例

```yml
- key: collect_doc_save
  module: model_save
  table: collect_doc
  params:
    create_time:
      template: "{{current_date_time}}"
    create_user:
      template: "{{.session_user_id}}"
    is_delete:
      default: "0"
```

## 注意事项

- 针对数据库表进行新增一行
- 可以利用params里面的template生成uuid、创建时间、创建人、设置默认时间
- params 下面一级是变量名称，比如示例中create_time 是表里面create_time 字段，下面生成规则，与校验规则
- 如果数据库表设置默认值，最好是参数先设置。因为gorm 设置默认值会报错

## 元信息

- 来源：`服务文档 -> 模块处理 -> 3.model_save / 表新增行`
- 页面标题：`model_save(表新增行)`
