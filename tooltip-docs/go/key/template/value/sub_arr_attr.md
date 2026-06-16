# template function=sub_arr_attr

## 作用

- 读取二维结构指定位置字段值。
- 可用于 `params[*].template`、`check.template`、`enable`、`if_template` 等模板表达式位置。

## 常见用途

- 在 `params.*.template` 中生成 `sub_arr_attr` 相关值。
- 在 `check.template`、`enable`、`if_template` 等条件表达式中复用。

## 执行阶段（低代码视角）

- 模板渲染阶段：在参数解析、校验判断、处理器模板计算时执行。

## 怎么用

### 参数
- 调用签名：`func SubArrAttr(arr []map[string]interface{}, x int, field string, y int, attr string)`
- 参数值来自当前服务上下文（例如 `.field`、`.item.field`、`.session_user_id`）。

### 配置位置
- `tooltip-docs/go/key/template/value/<function>.md`

## 示例

```yml
params:
  demo:
    template: '{{sub_arr_attr .rows 0 "children" 0 "id"}}'
```

## 注意事项

### 定位
- 悬停命中文档路径：`tooltip-docs/go/key/template/value/sub_arr_attr.md`
- 如需改行为，优先改实现文件，再更新本说明。

## 元信息

- 来源：
