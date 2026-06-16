# model_field

## 作用

- `model_field` 指定批量模块的数据来源字段（通常是数组字段名）。

## 常见用途

- `bulk_create`、`bulk_upsert` 等批量模块从参数池取列表数据。

## 执行阶段（低代码视角）

- 模块执行阶段：从参数池读取该字段作为待处理数据集。

## 怎么用

### 配置位置

- `service[].model_field`（批量模块）

## 示例

```yml
service:
  - key: mail_account_bulk_add
    module: bulk_create
    model_field: "[mail_account_list]"
```

来源文件：`/data/project/sport/collect/system/mail_account/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`model_field`
