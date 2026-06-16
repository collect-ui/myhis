# key=render_template

## 作用

- 读取模板文件内容，使用当前参数渲染，生成最终文本。
- 可将渲染结果保存到指定字段供后续流程使用。

## 常见用途

- 读取模板文件内容，使用当前参数渲染，生成最终文本。
- 可将渲染结果保存到指定字段供后续流程使用。

## 执行阶段（低代码视角）

- 参数处理阶段：在模块执行前运行，用于加工/校验/补充参数。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| template_file | string | 是 | 模板文件路径（按框架文件读取规则） |
| save_field | string | 否 | 渲染结果保存字段 |

## 示例

```yml
handler_params:
  - key: render_template
    template_file: publish_message.tpl
    save_field: publish_message
```

## 注意事项

- 渲染使用当前参数上下文（含上游处理器写入字段）。
- `save_field` 不配置时，只执行渲染不写字段。

### 源码定位
- Python类路径：`collect.service_imp.request_handlers.handlers.render_template.RenderTemplate`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/request_handlers/handlers/render_template.py`

## 元信息

- 来源：`handler_params -> render_template / 模板文件渲染`
- 页面标题：`render_template(模板文件渲染)`
