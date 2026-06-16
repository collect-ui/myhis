# default

## 作用

- 当请求未传该字段，或字段为空时，使用 `default` 填充默认值。
- 适合静态默认值（如 `is_delete=0`、`page=1`）。

## 常见用途

- 当请求未传该字段，或字段为空时，使用 `default` 填充默认值。
- 适合静态默认值（如 `is_delete=0`、`page=1`）。

## 执行阶段（低代码视角）

- 参数/处理器执行阶段：在参数计算、条件判断或处理器运行时生效。

## 怎么用

### 配置位置
- 请按字段所在节点配置。

## 示例

```yml
params:
  is_delete:
    default: 0
  page:
    default: 1
    type: int
```

## 注意事项

- `default` 是兜底值；若需要动态值（当前时间、当前用户），请用 `template`。
- 常见搭配：`default + type`（先给默认值，再做类型转换）。

## 元信息

- 来源：`服务文档 -> params 字段规则`
- 页面标题：`default(默认值)`
