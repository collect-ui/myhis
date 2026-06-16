# template function=base64ToImg

## 作用

- 将 base64 内容转换为图片结果。
- 用于 Python 低代码模板表达式（`template`、`check.template`、`enable`、`if_template`）。

## 常见用途

- 在 `params.*.template` 中生成 `base64ToImg` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 参数形式以实现类 `Base64ToImg` 的 `filter` 方法为准。
- 调用时可读取当前参数池中的字段（例如 `.item`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/python/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{base64ToImg .value}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/python/key/template/value/base64ToImg.md`
- 修改函数行为时，请同步更新 `filter_handler` 与本说明。

## 元信息

- 来源：
