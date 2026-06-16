# path

## 作用

- `path` 在不同上下文代表“文件路径”或“实现类路径”。

## 常见用途

- 项目路由里定位子配置文件；注册表里定位实现代码模块。

## 执行阶段（低代码视角）

- 配置解析阶段：先按 `path` 定位资源，再执行后续流程。

## 怎么用

### 配置位置

- `project/service.yml` 路由项
- `service_router.yml` 注册项

## 示例

```yml
  - key: 'amis_router'
    name: '项目路由'
    path: 'amis_router/service.yml'
  - key: 'config'
    name: 'config'
    path: 'config/service.yml'
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`path`
