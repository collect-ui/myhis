# check

## 作用

- `check` 用于定义参数校验规则。
- 通常搭配 `check.template + check.err_msg` 使用。
- 当校验模板返回 false 时，中断服务并返回错误信息。

## 常见用途

- `check` 用于定义参数校验规则。
- 通常搭配 `check.template + check.err_msg` 使用。

## 执行阶段（低代码视角）

- 参数/处理器执行阶段：在参数计算、条件判断或处理器运行时生效。

## 怎么用

### 配置位置
- 请按字段所在节点配置。

## 示例

```yml
params:
  doc_group_list:
    check:
      template: "{{must .doc_group_list}}"
      err_msg: 分组不能为空
```

## 注意事项

- `check.template` 建议返回明确布尔语义（True/False）。
- `check.err_msg` 支持模板，可拼接上下文字段（如行号、字段名）。
- 复杂校验建议先在 `handler_params` 里预处理再校验。

## 元信息

- 来源：`服务文档 -> params 字段规则`
- 页面标题：`check(参数校验)`
