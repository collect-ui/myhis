# module=bulk_create

## 作用

- 数据库表批量保存，bulk_service 针对服务类型批量操作，如果确定是同一张表，bulk_create创建数据，数据库连接只有一次，能减少数据库的连接次数
- bulk_service 是针对如何服务多线程跑
- 示例中的handler_params是参数处理模块
- 批量新增数据行

## 常见用途

- 数据库表批量保存，bulk_service 针对服务类型批量操作，如果确定是同一张表，bulk_create创建数据，数据库连接只有一次，能减少数据库的连接次数
- bulk_service 是针对如何服务多线程跑

## 执行阶段（低代码视角）

- 模块执行阶段：在 `handler_params` 完成后执行，是服务主体能力；执行结束后进入 `result_handler`。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | module: bulk_create |
| table | string | 是 | 数据库表名 |
| model_field | string | 是 | 取哪个列表字段。[你参数变量] |

## 示例

```yml
- key: config_detail_bulk_create
  name: 配置批量新增
  module: bulk_create
  log: true
  table: "config_detail"
  model_field: "[detail_list]"
  http: true
  params:
    detail_list:
      check:
        template: "{{must .detail_list}}"
        err_msg: 数据列表不能为空
  handler_params:
    - key: update_array
      foreach: "[detail_list]"
      item: item
      fields:
        - field: config_detail_id
          template: "{{uuid}}"
```

## 注意事项

- 批量新增数据行
- 利用handler_params 中update_array 可以对没行记录字段进行调整

## 元信息

- 来源：`服务文档 -> 模块处理 -> 6.bulk_create / 表批量保存`
- 页面标题：`bulk_create(表批量保存)`
