# data_json

## 作用

- `data_json` 指向 JSON 配置文件，常用于 HTTP 模块或 empty+渲染流程。

## 常见用途

- 把复杂请求体、模板数据、路由配置从 YAML 抽离到 JSON 文件。

## 执行阶段（低代码视角）

- 模块执行前：先读取并渲染 JSON，再交给模块/处理器消费。

## 怎么用

### 配置位置

- `service[].data_json`

## 示例

```yml
service:
  - key: system_login
    module: http
    data_json: system_login.json
```

来源文件：`/data/project/sport/collect/hrm/login/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`data_json`
