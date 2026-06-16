# must_token

## 作用

- `must_token: true` 表示该服务必须携带有效登录态/令牌才能访问。

## 常见用途

- 对外 HTTP 服务做访问保护，防止匿名调用。

## 执行阶段（低代码视角）

- HTTP 入口鉴权阶段：在进入业务服务前先校验 token。

## 怎么用

### 配置位置

- `service[].must_token`

## 示例

```yml
    module: ssh
    http: true
    must_token: true
    check_ip: true
    params:
      timeout:
```

来源文件：`/data/project/moongod-backend/backend_data_service/release/deploy/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`must_token`
