# template function=unix_time2datetime

## 作用

- Unix 时间戳转标准日期时间字符串。
- 可用于 `params[*].template`、`check.template`、`enable`、`if_template` 等模板表达式位置。

## 常见用途

- 在 `params.*.template` 中生成 `unix_time2datetime` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 调用签名：`func UnixTime2Datetime(unit int64)`
- 参数值来自当前服务上下文（例如 `.field`、`.item.field`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/go/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{unix_time2datetime .item.checkin_time}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/go/key/template/value/unix_time2datetime.md`
- 如需改行为，优先改实现文件，再更新本说明。

## 元信息

- 来源：
