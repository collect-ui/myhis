# data_handler

## 作用

- `data_handler` 是服务路由中的数据处理器注册表，声明可用处理器实现。

## 常见用途

- 维护框架级“字段转换/文件加载/映射”等公共处理器。

## 执行阶段（低代码视角）

- 引擎启动与路由加载阶段：用于能力注册，不是业务服务字段。

## 怎么用

### 配置位置

- `service_router.yml` 根级 `data_handler:`

## 示例

```yml
data_handler:
  - key: update_field
    name: 添加参数
    type: inner
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`data_handler`
