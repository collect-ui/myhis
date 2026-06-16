# key=str2img

## 作用

- 把参数中的 Base64 字符串解码为图片文件并写入磁盘。
- 支持 `data:image/png;base64,...` 这种带前缀的数据，也支持纯 Base64 字符串。
- 常用于图片上传后落盘，再把落盘路径写回业务表。

## 常见用途

- 把参数中的 Base64 字符串解码为图片文件并写入磁盘。
- 支持 `data:image/png;base64,...` 这种带前缀的数据，也支持纯 Base64 字符串。

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| key | string | 是 | 固定为 `str2img` |
| value | string/template | 是 | Base64 图片内容，支持模板变量，如 `[img]` |
| path | string/template | 是 | 输出图片完整路径，支持模板变量，如 `[local_path]` |

## 示例

```yml
handler_params:
  - key: update_field
    fields:
      - field: local_path
        template: "{{(get_key `local_file_dir`)}}/doc/img/{{sub_str .create_time 0 10}}/{{.img_id}}.png"
  - key: str2img
    value: "[img]"
    path: "[local_path]"
```

## 注意事项

- `value` 不是字符串时会报错：`content must be a base64 string`。
- `path` 为空会报错：`output image path is required`。
- 自动去掉 `base64,` 之前的前缀（如 Data URL 头）。
- 自动创建目标目录（不存在时 `mkdir -p`）。
- 文件写入成功后返回成功信息，返回值包含实际写入路径。

### 源码定位
- Go 实现：`/data/project/collect/src/collect/service_imp/handler_params_str2img.go`
- 注册位置：`/data/project/sport/collect/service_router.yml` 的 `data_handler` -> `key: str2img`
- 典型使用：`/data/project/sport/collect/doc/img/index.yml`

## 元信息

- 来源：`服务文档 -> 参数处理 -> str2img / Base64字符串转图片`
- 页面标题：`str2img(Base64字符串转图片)`
