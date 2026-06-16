# python template functions

## 作用

- `index` 是模板函数。

## 常见用途

- 在 `params.*.template` 中生成 `index` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 以函数签名与实现为准。

### 配置位置
- `tooltip-docs/python/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{index}}'
```

## 注意事项

- 以当前项目实际注册函数和源码实现为准。

## 元信息

- 该目录存放 Python 低代码模板函数悬停文档。
- 规则：`tooltip-docs/python/key/template/value/<function>.md`
- 来源：`/data/project/moongod-backend/backend_data_service/service_router.yml` 的 `filter_handler`
- 维护建议：新增/替换 filter_handler 后同步补文档。
- 来源：`模板函数目录索引`
- 页面标题：`index`
