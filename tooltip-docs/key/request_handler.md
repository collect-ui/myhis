# request_handler

## 作用

- `request_handler` 是请求处理器注册表，声明 `handler_params` 可用处理器。

## 常见用途

- 新增参数处理器（模块前处理）时在此注册。

## 执行阶段（低代码视角）

- 引擎启动加载阶段：注册请求处理能力。

## 怎么用

### 配置位置

- `service_router.yml` 根级 `request_handler:`

## 示例

```yml

# 请求处理器
request_handler:
  # 去重
  - key: distinct
    path: collect.service_imp.request_handlers.handlers.distinct
```

来源文件：`/data/project/moongod-backend/backend_data_service/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`request_handler`
