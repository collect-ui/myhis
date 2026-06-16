# key=file2datajson

## 作用

- 读取并渲染当前服务的 `data_json` 配置文本，然后反序列化为对象或数组。
- 支持在 JSON 字符串中使用 `{{to_json .xxx}}` 把复杂对象嵌入为合法 JSON。

## 常见用途

- 读取并渲染当前服务的 `data_json` 配置文本，然后反序列化为对象或数组。
- 支持在 JSON 字符串中使用 `{{to_json .xxx}}` 把复杂对象嵌入为合法 JSON。

## 执行阶段（低代码视角）

- 参数处理阶段：在模块执行前运行，用于加工/校验/补充参数。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | 固定为 `file2datajson` |
| save_original | bool | 否 | 为 true 时不反序列化，直接返回渲染后的原始文本 |
| save_field | string | 否 | 保存到 params 的字段（若框架通用逻辑启用） |

## 示例

```yml
- key: project_router
  module: empty
  data_json: project_router.json
  handler_params:
    - key: file2datajson
      save_field: data
    - key: param2result
      field: data
```

## 注意事项

- 先做模板渲染，再处理 `to_json` 包装占位符。
- 先按对象解析；若对象解析为空，再尝试按数组解析。
- 解析失败时会返回空对象/空数组语义，建议配合模板校验。

### 源码定位
- Go实现：`/data/project/collect/src/collect/service_imp/handler_params_file2datajson.go`

## 元信息

- 来源：`handler_params -> file2datajson / 配置文件转JSON对象`
- 页面标题：`file2datajson(配置文件转JSON对象)`
