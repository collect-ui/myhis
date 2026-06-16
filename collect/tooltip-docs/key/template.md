# template

模板表达式字段。运行时会根据上下文变量渲染。

## 常见用途
- 模糊查询拼接
- 时间范围拼接
- 条件启用表达式

## 示例
```yml
template: "{% if search %}%{{search}}%{% endif %}"
```
