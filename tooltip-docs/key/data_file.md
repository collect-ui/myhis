# data_file

## 作用

- `data_file` 指向外部配置/脚本文件，模块运行前会先读取该文件内容。

## 常见用途

- SQL 文件引用、LDAP 配置、模板文件、规则文件等外置化管理。

## 执行阶段（低代码视角）

- 模块执行前：先加载文件内容，再进入模块逻辑。

## 怎么用

### 配置位置

- `service[].data_file`
- `service[].data_json` 相关处理流程

## 示例

```yml
    http: true
    module: sql
    data_file: work_task_project_query.sql
```

来源文件：`/data/project/sport/collect/work_task/work_task_project/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`data_file`
