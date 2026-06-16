# template function=sub_str

## 作用

- 按开始/结束位置截取字符串，支持负索引。
- 可用于 `params[*].template`、`check.template`、`enable`、`if_template` 等模板表达式位置。

## 常见用途

- 在 `params.*.template` 中生成 `sub_str` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 调用签名：`func SubStr(content string, start int, end int)`
- 参数值来自当前服务上下文（例如 `.field`、`.item.field`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/go/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{sub_str .code 0 8}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/go/key/template/value/sub_str.md`
- 如需改行为，优先改实现文件，再更新本说明。

## 元信息

- 来源：
