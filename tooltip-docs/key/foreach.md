# foreach

## 作用

- `foreach` 指定要循环处理的数组来源（参数字段或模板表达式）。

## 常见用途

- 数组逐项更新、数组过滤、批量服务调用、批量校验。

## 执行阶段（低代码视角）

- 参数处理或结果处理阶段：处理器执行时按 `foreach` 取数组并循环。

## 怎么用

### 配置位置

- 处理器节点下（如 `update_array/filter_arr/check_array/bulk_service`）

## 示例

```yml
handler_params:
  - key: update_array
    foreach: "[service_list]"
    item: item
```

来源文件：`/data/project/sport/collect/work_task/bulk/index.yml`

## 注意事项

- 字段取值建议与业务语义保持一致，避免仅用临时命名。
- 跨 Python/Go 项目时，请先确认同名字段在当前引擎中的实现差异。

## 元信息

- 来源：`低代码关键字通用说明`
- 页面标题：`foreach`
