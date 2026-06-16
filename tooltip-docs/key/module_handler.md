# module_handler

## 作用

- `module_handler` 是模块注册表，定义 `module` 值到实现代码的映射。

## 常见用途

- 新增模块能力时在此注册，供服务 `module: xxx` 调用。

## 执行阶段（低代码视角）

- 引擎启动加载阶段：注册模块能力。

## 怎么用

### 配置位置

- `service_router.yml` 根级 `module_handler:`

## 示例

```yml

# 模块处理器
module_handler:
  # 模型修改
  - key: sql
    name: 执行sql 查询
```

来源文件：`/data/project/sport/collect/service_router.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`module_handler`
