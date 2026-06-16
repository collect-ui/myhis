# template function=html2jira_text

## 作用

- HTML 转 Jira 文本格式。
- 用于 Python 低代码模板表达式（`template`、`check.template`、`enable`、`if_template`）。

## 常见用途

- 在 `params.*.template` 中生成 `html2jira_text` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 参数形式以实现类 `Html2JiraText` 的 `filter` 方法为准。
- 调用时可读取当前参数池中的字段（例如 `.item`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/python/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{html2jira_text .html}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/python/key/template/value/html2jira_text.md`
- 修改函数行为时，请同步更新 `filter_handler` 与本说明。

## 元信息

- 来源：
