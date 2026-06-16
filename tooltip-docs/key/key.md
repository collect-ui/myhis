# key

## 作用

- `key` 是低代码最核心字段：用于定义服务名、处理器名、模块名。

## 常见用途

- 服务定义 `service[].key`；处理器定义 `handler_params[].key`；注册表定义 `*.key`。

## 执行阶段（低代码视角）

- 配置解析与执行分发阶段：引擎根据 `key` 决定调用哪个逻辑。

## 怎么用

### 配置位置

- 服务节点、处理器节点、路由注册节点

## 示例

```yml
    http_json: mini_app_token.json
    cache:
      key: "handler_cache"
      enable: "{{eq (get_key \"can_cache\") \"true\"}}"
      room: wechat
      second: 600
```

来源文件：`/data/project/sport/collect/wechat/mini_app_token/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`key`
