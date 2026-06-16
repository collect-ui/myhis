# table

## 作用

- `table` 指定数据库目标表，供增删改模块操作。

## 常见用途

- `model_save/model_update/model_delete/bulk_create/bulk_upsert` 等模块。

## 执行阶段（低代码视角）

- 模块执行阶段：根据 `table` 构造 SQL/ORM 操作。

## 怎么用

### 配置位置

- `service[].table`

## 示例

```yml
service:
  - key: work_task_version_save
    module: model_save
    table: work_task_version
```

来源文件：`/data/project/sport/collect/work_task/version/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`table`
