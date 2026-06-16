# class_name

## 作用

- `class_name` 是处理器注册项的实现类名，和 `path`、`method` 一起决定运行哪个代码。

## 常见用途

- 在 `service_router.yml` 的注册表中声明模块、请求处理器、结果处理器、模板函数实现。

## 执行阶段（低代码视角）

- 引擎启动与配置加载阶段：读取注册表时解析，不在单次业务请求中动态计算。

## 怎么用

### 配置位置

- `module_handler[]`
- `request_handler[]`
- `result_handler[]`
- `filter_handler[]`
- `key_word_rules.*`

## 示例

```yml
module_handler:
  - key: sql
    path: collect.service_imp.sql.sql_service
    class_name: SQLService
    method: handler
```

来源文件：`/data/project/moongod-backend/backend_data_service/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`class_name`
