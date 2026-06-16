# key=file_response

## 作用

- 把本次结果转换为“文件响应对象”，用于下载文件。

## 常见用途

- 把本次结果转换为“文件响应对象”，用于下载文件。

## 执行阶段（低代码视角）

- 结果处理阶段：在模块执行后运行，用于重组输出或联动后续动作。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| params.path | string/template | 是 | 文件路径或路径变量 |
| params.filename | string/template | 否 | 下载文件名；不配默认取路径 basename |

## 示例

```yml
result_handler:
  - key: file_response
    params:
      path: export_path
      filename: "{{.export_name}}.xlsx"
```

## 注意事项

- 会校验文件是否存在，不存在直接报错。
- `path` 和 `filename` 都支持模板渲染。

### 源码定位
- Python类路径：`collect.service_imp.result_handlers.handlers.file_response.FileResponse`
- 本次核对源码：`/tmp/collect-wheel-0.0.86/collect/service_imp/result_handlers/handlers/file_response.py`

## 元信息

- 来源：`result_handler -> file_response / 文件下载响应`
- 页面标题：`file_response(文件下载响应)`
