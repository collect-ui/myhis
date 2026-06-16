# require

## 作用

- `require` 用于引入外部配置文件内容并参与当前配置渲染。

## 常见用途

- 公共 SQL 片段、公共 HTTP 模板、共享字段配置复用。

## 执行阶段（低代码视角）

- 配置加载阶段：先展开 `require`，再进行模板渲染与执行。

## 怎么用

### 配置位置

- SQL/JSON/YAML 配置中（按引擎支持）

## 示例

```yml
# SQL 模板关键字规则
key_word_rules:
  require:
    path: collect.service_imp.key_word_rules.require
    class_name: Require
    reg: require[(](.*?)[)]
```

来源文件：`/data/project/moongod-backend/backend_data_service/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`require`
