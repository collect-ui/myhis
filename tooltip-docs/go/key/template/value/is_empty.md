# template function=is_empty

## 作用

- 判断值是否为空（空串/空数组/nil 等）。
- 可用于 `params[*].template`、`check.template`、`enable`、`if_template` 等模板表达式位置。

## 常见用途

- 在 `params.*.template` 中生成 `is_empty` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 调用签名：`func IsEmpty(value interface{})`
- 参数值来自当前服务上下文（例如 `.field`、`.item.field`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/go/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{is_empty .remark}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/go/key/template/value/is_empty.md`
- 如需改行为，优先改实现文件，再更新本说明。

## 元信息

- 来源：
